/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  
  // Allow external API calls
  async rewrites() {
    return [
      {
        source: '/api/sandbox/:path*',
        destination: process.env.SANDBOX_EXECUTOR_URL || 'http://sandbox-executor.quantumlayer.svc.cluster.local:8085/api/:path*'
      },
      {
        source: '/api/capsule/:path*',
        destination: process.env.CAPSULE_BUILDER_URL || 'http://capsule-builder.quantumlayer.svc.cluster.local:8086/api/:path*'
      },
      {
        source: '/api/drops/:path*',
        destination: process.env.QUANTUM_DROPS_URL || 'http://quantum-drops.temporal.svc.cluster.local:8080/api/:path*'
      }
    ]
  },
  
  // Environment variables
  env: {
    NEXT_PUBLIC_APP_NAME: 'QuantumLayer Preview',
    NEXT_PUBLIC_API_URL: process.env.API_URL || 'http://localhost:3000'
  },

  // Monaco Editor webpack configuration
  webpack: (config) => {
    config.resolve.fallback = {
      ...config.resolve.fallback,
      fs: false,
    };
    return config;
  }
}

module.exports = nextConfig