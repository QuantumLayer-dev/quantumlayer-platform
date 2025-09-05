/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  env: {
    QINFRA_API_URL: process.env.QINFRA_API_URL || 'http://localhost:8095',
    QINFRA_AI_API_URL: process.env.QINFRA_AI_API_URL || 'http://localhost:8098',
    IMAGE_REGISTRY_API_URL: process.env.IMAGE_REGISTRY_API_URL || 'http://localhost:30096',
  },
  async rewrites() {
    return [
      {
        source: '/api/qinfra/:path*',
        destination: `${process.env.QINFRA_API_URL || 'http://localhost:8095'}/:path*`,
      },
      {
        source: '/api/ai/:path*',
        destination: `${process.env.QINFRA_AI_API_URL || 'http://localhost:8098'}/api/v1/:path*`,
      },
    ]
  },
}

module.exports = nextConfig