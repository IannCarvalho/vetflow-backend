#!/bin/bash
set -e

echo "[ENTRYPOINT] Starting application..."
echo "[ENTRYPOINT] Working directory: $(pwd)"
echo "[ENTRYPOINT] Binary exists: $(test -f ./main && echo 'YES' || echo 'NO')"
echo "[ENTRYPOINT] Binary executable: $(test -x ./main && echo 'YES' || echo 'NO')"
echo "[ENTRYPOINT] Binary info: $(ls -lh ./main 2>&1 || echo 'NOT FOUND')"
echo "[ENTRYPOINT] MONGO_URI set: $(test -n "$MONGO_URI" && echo 'YES' || echo 'NO')"
echo "[ENTRYPOINT] PORT set: $(test -n "$PORT" && echo 'YES' || echo 'NO')"

echo "[ENTRYPOINT] Executing main binary..."
exec ./main "$@"
