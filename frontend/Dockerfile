FROM node:25-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_APP_VERSION=dev
ENV VITE_APP_VERSION=$VITE_APP_VERSION
RUN npm run build

FROM node:25-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/build ./build
COPY --from=builder /app/package*.json ./
RUN npm ci --production --ignore-scripts
EXPOSE 3000
CMD ["node", "build"]
