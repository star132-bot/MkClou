import type { NextConfig } from "next";

// 后端 API 地址。前端统一请求同源的 /api/v1/*，由这里转发到 Go 服务，
// 避免跨域并让 Refresh Token Cookie 属于前端域名；生产环境由 Nginx 完成同样的转发。
const apiOrigin = process.env.API_ORIGIN ?? "http://127.0.0.1:18080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/v1/:path*", destination: `${apiOrigin}/api/v1/:path*` }];
  },
};

export default nextConfig;
