FROM node:20-alpine AS commons-builder
COPY commons /app/commons
WORKDIR /app/commons/ts
RUN npm install
RUN npm run build

FROM node:20-alpine
ARG CI_VERSION
ENV NEXT_PUBLIC_PRODUCT_VERSION=$CI_VERSION
ENV NODE_ENV=development
COPY --from=commons-builder /app/commons/ts/ /app/commons/ts
ADD booking-ui /app/
WORKDIR /app
RUN npm install
RUN npm install --save ./commons/ts
EXPOSE 3001
# Workaround for these bugs:
# https://github.com/vercel/next.js/issues/51684
# https://github.com/tiredofit/docker-collabora-online/pull/28
# RUN sed -i "s/const hostname = /const hostname = process.env.LISTEN_ADDR || /g" /app/build/standalone/server.js
ENTRYPOINT ["/usr/local/bin/npm", "run", "dev"]
