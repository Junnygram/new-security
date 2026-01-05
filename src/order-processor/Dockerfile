# Build Stage
FROM python:3.11-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Run Stage
FROM python:3.11-slim
WORKDIR /app
# Copy installed packages from builder or just install them here. 
# Since slim includes pip, we can just install here or copy site-packages.
# Simpler for this context to just install in one stage or copy.
# Let's use a clean install in the final stage to avoid copying complexity with system libs.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py .

# Create a non-root user for security (mimicking distroless nonroot)
RUN useradd -m nonroot && chown -R nonroot /app
USER nonroot

ENTRYPOINT ["python", "app.py"]
