# 🌮 Taco Spin Service

This Go web service simulates a taco spinning in the wind of Chicago. The spin rate is derived from the current wind speed using a windmill RPM approximation.

## Features
- `/start`: Starts tracking taco spins
- `/stop`: Stops tracking
- `/spins`: Returns total spins since start

## Setup
1. Create a `.env` file or set `OPENWEATHER_API_KEY` in the environment.
2. Build and run:
   ```bash
   docker-compose up --build
   ```

## Sample API usage
```bash
curl -X POST localhost:8080/start
curl localhost:8080/spins
curl -X POST localhost:8080/stop
```

Enjoy your wind-powered taco! 🌮💨

