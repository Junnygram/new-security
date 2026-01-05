/** @type {import('next').NextConfig} */
const nextConfig = {
    output: 'standalone',
    async rewrites() {
        return [
            {
                source: '/api/:path*',
                destination: 'http://store-api:8080/:path*',
            },
        ]
    },
}
module.exports = nextConfig
