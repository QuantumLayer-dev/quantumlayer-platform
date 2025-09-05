#!/bin/bash
set -e

# CIS Ubuntu 22.04 Hardening Script
# Implements CIS Benchmark security controls
# Version: 1.0.0

echo "========================================="
echo "Starting CIS Hardening for Ubuntu 22.04"
echo "========================================="

# 1. Filesystem Configuration
echo "[*] Configuring filesystem..."

# 1.1 Disable unused filesystems
cat >> /etc/modprobe.d/cis-hardening.conf << EOF
# CIS 1.1.1 Disable unused filesystems
install cramfs /bin/true
install freevxfs /bin/true
install jffs2 /bin/true
install hfs /bin/true
install hfsplus /bin/true
install squashfs /bin/true
install udf /bin/true
install vfat /bin/true
EOF

# 1.2 Configure /tmp
echo "tmpfs /tmp tmpfs defaults,rw,nosuid,nodev,noexec,relatime,size=2G 0 0" >> /etc/fstab

# 1.3 Configure separate partitions (if applicable)
# Note: This would be done during initial provisioning

# 2. Configure Software Updates
echo "[*] Configuring automatic updates..."
apt-get install -y unattended-upgrades
dpkg-reconfigure -plow unattended-upgrades

cat > /etc/apt/apt.conf.d/50unattended-upgrades << EOF
Unattended-Upgrade::Allowed-Origins {
    "\${distro_id}:\${distro_codename}-security";
    "\${distro_id}ESMApps:\${distro_codename}-apps-security";
    "\${distro_id}ESM:\${distro_codename}-infra-security";
};
Unattended-Upgrade::AutoFixInterruptedDpkg "true";
Unattended-Upgrade::MinimalSteps "true";
Unattended-Upgrade::Remove-Unused-Dependencies "true";
Unattended-Upgrade::Automatic-Reboot "false";
EOF

# 3. Filesystem Integrity
echo "[*] Setting up AIDE..."
aideinit -y -f

# 4. Secure Boot Settings
echo "[*] Securing boot settings..."
chown root:root /boot/grub/grub.cfg
chmod og-rwx /boot/grub/grub.cfg

# 5. Process Hardening
echo "[*] Configuring process hardening..."

# 5.1 Enable ASLR
sysctl -w kernel.randomize_va_space=2

# 5.2 Restrict core dumps
echo "* hard core 0" >> /etc/security/limits.conf
sysctl -w fs.suid_dumpable=0

# 6. Network Parameters
echo "[*] Hardening network parameters..."

cat >> /etc/sysctl.d/99-cis-network.conf << EOF
# IP Forwarding
net.ipv4.ip_forward = 0
net.ipv6.conf.all.forwarding = 0

# Send redirects
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0

# Source routed packets
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.conf.default.accept_source_route = 0
net.ipv6.conf.all.accept_source_route = 0
net.ipv6.conf.default.accept_source_route = 0

# ICMP redirects
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv6.conf.all.accept_redirects = 0
net.ipv6.conf.default.accept_redirects = 0

# Secure ICMP redirects
net.ipv4.conf.all.secure_redirects = 0
net.ipv4.conf.default.secure_redirects = 0

# Log Martians
net.ipv4.conf.all.log_martians = 1
net.ipv4.conf.default.log_martians = 1

# Ignore ICMP ping
net.ipv4.icmp_echo_ignore_broadcasts = 1

# Ignore bogus error responses
net.ipv4.icmp_ignore_bogus_error_responses = 1

# SYN cookies
net.ipv4.tcp_syncookies = 1

# Reverse path filtering
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# IPv6 router advertisements
net.ipv6.conf.all.accept_ra = 0
net.ipv6.conf.default.accept_ra = 0
EOF

sysctl -p /etc/sysctl.d/99-cis-network.conf

# 7. Configure UFW Firewall
echo "[*] Configuring firewall..."
ufw --force enable
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp  # SSH
ufw allow 443/tcp # HTTPS
ufw allow 9100/tcp # Node Exporter
ufw logging on

# 8. Configure Auditd
echo "[*] Configuring audit system..."

cat >> /etc/audit/rules.d/cis.rules << EOF
# CIS Audit Rules

# 4.1.3 Changes to system administration scope
-w /etc/sudoers -p wa -k scope
-w /etc/sudoers.d/ -p wa -k scope

# 4.1.4 Login and logout events
-w /var/log/faillog -p wa -k logins
-w /var/log/lastlog -p wa -k logins
-w /var/log/tallylog -p wa -k logins

# 4.1.5 Session initiation
-w /var/run/utmp -p wa -k session
-w /var/log/wtmp -p wa -k logins
-w /var/log/btmp -p wa -k logins

# 4.1.6 Discretionary access controls
-a always,exit -F arch=b64 -S chmod -S fchmod -S fchmodat -F auid>=1000 -F auid!=4294967295 -k perm_mod
-a always,exit -F arch=b64 -S chown -S fchown -S fchownat -S lchown -F auid>=1000 -F auid!=4294967295 -k perm_mod
-a always,exit -F arch=b64 -S setxattr -S lsetxattr -S fsetxattr -S removexattr -F auid>=1000 -F auid!=4294967295 -k perm_mod

# 4.1.7 Unsuccessful file access
-a always,exit -F arch=b64 -S open -S openat -S truncate -S ftruncate -F exit=-EACCES -F auid>=1000 -F auid!=4294967295 -k access
-a always,exit -F arch=b64 -S open -S openat -S truncate -S ftruncate -F exit=-EPERM -F auid>=1000 -F auid!=4294967295 -k access

# 4.1.8 Privileged commands
-a always,exit -F path=/usr/bin/passwd -F perm=x -F auid>=1000 -F auid!=4294967295 -k privileged
-a always,exit -F path=/usr/bin/sudo -F perm=x -F auid>=1000 -F auid!=4294967295 -k privileged

# 4.1.9 Successful file system mounts
-a always,exit -F arch=b64 -S mount -F auid>=1000 -F auid!=4294967295 -k mounts

# 4.1.10 File deletion
-a always,exit -F arch=b64 -S unlink -S unlinkat -S rename -S renameat -F auid>=1000 -F auid!=4294967295 -k delete

# Make configuration immutable
-e 2
EOF

systemctl restart auditd

# 9. Configure PAM
echo "[*] Hardening PAM configuration..."

# 9.1 Password quality
cat > /etc/security/pwquality.conf << EOF
# Password Quality Configuration
minlen = 14
dcredit = -1
ucredit = -1
ocredit = -1
lcredit = -1
EOF

# 9.2 Lockout policy
cat >> /etc/pam.d/common-auth << EOF
# Account lockout after 5 failed attempts
auth required pam_tally2.so onerr=fail audit silent deny=5 unlock_time=900
EOF

# 9.3 Password reuse
cat >> /etc/pam.d/common-password << EOF
# Remember last 5 passwords
password required pam_pwhistory.so remember=5
EOF

# 10. User Accounts and Environment
echo "[*] Configuring user accounts..."

# 10.1 Set password expiry
sed -i 's/^PASS_MAX_DAYS.*/PASS_MAX_DAYS   90/' /etc/login.defs
sed -i 's/^PASS_MIN_DAYS.*/PASS_MIN_DAYS   7/' /etc/login.defs
sed -i 's/^PASS_WARN_AGE.*/PASS_WARN_AGE   7/' /etc/login.defs

# 10.2 Set umask
echo "umask 027" >> /etc/bash.bashrc
echo "umask 027" >> /etc/profile

# 10.3 Disable inactive accounts
useradd -D -f 30

# 11. Configure SSH
echo "[*] Hardening SSH configuration..."

cat > /etc/ssh/sshd_config.d/cis-hardening.conf << EOF
# CIS SSH Hardening
Protocol 2
LogLevel VERBOSE
X11Forwarding no
MaxAuthTries 4
IgnoreRhosts yes
HostbasedAuthentication no
PermitRootLogin no
PermitEmptyPasswords no
PermitUserEnvironment no
Ciphers chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-gcm@openssh.com,aes256-gcm@openssh.com
MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512,hmac-sha2-256
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,diffie-hellman-group14-sha256,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512,ecdh-sha2-nistp521,ecdh-sha2-nistp384,ecdh-sha2-nistp256
ClientAliveInterval 300
ClientAliveCountMax 0
LoginGraceTime 60
Banner /etc/issue.net
MaxStartups 10:30:60
MaxSessions 4
EOF

systemctl restart sshd

# 12. Configure Legal Banner
echo "[*] Setting legal banner..."

cat > /etc/issue.net << EOF
###############################################################
#                                                             #
#  This system is for authorized use only. By accessing      #
#  this system, you agree that your actions may be           #
#  monitored and recorded.                                   #
#                                                             #
#  Unauthorized access is strictly prohibited and will       #
#  be prosecuted to the fullest extent of the law.          #
#                                                             #
###############################################################
EOF

cp /etc/issue.net /etc/issue

# 13. Remove unnecessary packages
echo "[*] Removing unnecessary packages..."
apt-get -y purge xinetd nis yp-tools tftpd atftpd tftpd-hpa telnetd rsh-server rsh-redone-server 2>/dev/null || true

# 14. Ensure permissions on important files
echo "[*] Setting secure file permissions..."
chmod 644 /etc/passwd
chmod 640 /etc/shadow
chmod 644 /etc/group
chmod 640 /etc/gshadow
chmod 644 /etc/passwd-
chmod 640 /etc/shadow-
chmod 644 /etc/group-
chmod 640 /etc/gshadow-

# 15. Enable AppArmor
echo "[*] Enabling AppArmor..."
systemctl enable apparmor
systemctl start apparmor

# 16. Configure log rotation
echo "[*] Configuring log rotation..."
cat > /etc/logrotate.d/rsyslog << EOF
/var/log/syslog
/var/log/mail.info
/var/log/mail.warn
/var/log/mail.err
/var/log/mail.log
/var/log/daemon.log
/var/log/kern.log
/var/log/auth.log
/var/log/user.log
/var/log/lpr.log
/var/log/cron.log
/var/log/debug
/var/log/messages
{
    rotate 4
    weekly
    missingok
    notifempty
    compress
    delaycompress
    sharedscripts
    postrotate
        /usr/lib/rsyslog/rsyslog-rotate
    endscript
}
EOF

# 17. Create compliance report
echo "[*] Generating compliance report..."
mkdir -p /var/log/compliance

cat > /var/log/compliance/cis-hardening-report.txt << EOF
CIS Hardening Report
Generated: $(date)
Hostname: $(hostname)
OS: Ubuntu 22.04

Hardening Steps Completed:
✓ Filesystem configuration
✓ Automatic updates configured
✓ AIDE installed and initialized
✓ Boot settings secured
✓ Process hardening enabled
✓ Network parameters hardened
✓ UFW firewall configured
✓ Auditd rules installed
✓ PAM hardened
✓ User account policies set
✓ SSH hardened
✓ Legal banners configured
✓ Unnecessary packages removed
✓ File permissions secured
✓ AppArmor enabled
✓ Log rotation configured

Compliance Status: HARDENED
EOF

echo "========================================="
echo "CIS Hardening Complete!"
echo "Report saved to: /var/log/compliance/cis-hardening-report.txt"
echo "========================================="