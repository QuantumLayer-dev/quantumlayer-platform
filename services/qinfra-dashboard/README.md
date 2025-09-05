# QInfra Dashboard

Enterprise Infrastructure Resilience Platform - Single Pane of Glass

## Features Implemented

### 🎯 Executive Dashboard
- **Real-time KPIs**: Health score, vulnerabilities, compliance, patch success rate
- **Infrastructure Drift Heatmap**: Visual representation of 847+ monitored nodes
- **Compliance Gauge**: SOC2, HIPAA, PCI-DSS, ISO 27001 tracking
- **Patch Timeline**: Live deployment status and scheduling
- **Risk Assessment Matrix**: 5x5 probability vs impact grid

### 🤖 AI Intelligence Integration
- **Drift Prediction**: ML-powered drift forecasting
- **Patch Risk Scoring**: Automated risk assessment for patches
- **Anomaly Detection**: Real-time infrastructure anomaly alerts
- **Remediation Advisor**: AI-powered fix recommendations
- **Canary Analysis**: Intelligent deployment validation

### 📊 Real-time Features
- **WebSocket Updates**: Live metrics and alerts
- **Toast Notifications**: Instant drift, patch, and compliance alerts
- **Auto-refresh**: 30-second metric updates
- **Connection Status**: Live/offline indicator

## Tech Stack

- **Framework**: Next.js 14 with TypeScript
- **UI Components**: Radix UI, Framer Motion
- **State Management**: Zustand
- **Real-time**: Socket.io Client
- **Charts**: Recharts, Chart.js
- **Styling**: Tailwind CSS
- **Icons**: Lucide React

## Quick Start

```bash
# Install dependencies
npm install

# Development
npm run dev

# Build for production
npm run build

# Start production server
npm start
```

## Environment Variables

```env
QINFRA_API_URL=http://localhost:8095
QINFRA_AI_API_URL=http://localhost:8098
IMAGE_REGISTRY_API_URL=http://localhost:30096
NEXT_PUBLIC_WS_URL=http://localhost:8099
```

## Docker Deployment

```bash
# Build image
docker build -t qinfra-dashboard .

# Run container
docker run -p 3003:3003 \
  -e QINFRA_API_URL=http://qinfra-api:8095 \
  -e QINFRA_AI_API_URL=http://qinfra-ai:8098 \
  qinfra-dashboard
```

## Dashboard Views

### 1. Executive Dashboard (/)
- Infrastructure health overview
- Critical vulnerability tracking
- Compliance score monitoring
- Patch success metrics
- Golden image inventory
- MTTR and RTO achievement
- Security posture assessment

### 2. Operations Center (Coming Soon)
- Live infrastructure monitoring
- Active incident management
- Resource utilization graphs
- Network topology view

### 3. Compliance Hub (Coming Soon)
- Framework-specific dashboards
- Control validation status
- Audit trail visualization
- Evidence collection tracking

## API Integration

The dashboard integrates with:

1. **QInfra API** (port 8095)
   - Golden image management
   - Patch orchestration
   - Drift detection

2. **QInfra AI API** (port 8098)
   - Predictive analytics
   - Risk assessment
   - Anomaly detection

3. **Image Registry API** (port 30096)
   - Image inventory
   - Vulnerability scanning
   - Compliance validation

## Component Architecture

```
src/
├── app/
│   ├── layout.tsx         # Root layout with dark theme
│   ├── page.tsx          # Executive Dashboard
│   └── globals.css       # Global styles
├── components/
│   ├── MetricCard.tsx    # KPI display cards
│   ├── DriftHeatmap.tsx  # Node drift visualization
│   ├── ComplianceGauge.tsx # Compliance scoring
│   ├── PatchTimeline.tsx # Patch deployment timeline
│   └── RiskMatrix.tsx    # Risk assessment grid
├── hooks/
│   └── useWebSocket.tsx  # Real-time connection hook
└── lib/
    └── store.ts         # Zustand state management
```

## Performance Optimizations

- **Code Splitting**: Automatic with Next.js
- **Image Optimization**: Next/Image for assets
- **Lazy Loading**: Components loaded on demand
- **Memoization**: React.memo for expensive renders
- **Virtual Scrolling**: For large data sets

## Security Features

- **CSP Headers**: Content Security Policy
- **HTTPS Only**: Enforced in production
- **Input Sanitization**: All user inputs validated
- **API Authentication**: Token-based auth ready
- **Rate Limiting**: Built-in with Next.js

## Monitoring & Observability

- **Error Tracking**: Console logging (Sentry ready)
- **Performance Metrics**: Web Vitals tracking
- **User Analytics**: Ready for GA/Mixpanel
- **Health Endpoints**: /api/health for monitoring

## Browser Support

- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions
- Mobile: iOS Safari, Chrome Android

## License

Proprietary - QuantumLayer Platform