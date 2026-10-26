FROM node:22-bookworm-slim AS build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
COPY VERSION /src/VERSION
RUN npm run build

FROM node:22-bookworm-slim
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0
COPY --from=build /src/web/build ./build
COPY --from=build /src/web/package.json ./package.json
EXPOSE 3000
CMD ["node", "build/index.js"]
