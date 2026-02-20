FROM node:20-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install --omit=dev

COPY . .
RUN mkdir -p /app/data

ENV PORT=3000
ENV RPC_PORT=50051
ENV DATA_FILE=/app/data/modeloman-db.json

EXPOSE 3000
EXPOSE 50051

CMD ["npm", "start"]
