FROM python:3.11-alpine

WORKDIR /app

# Copy the web UI files
COPY index.html .
COPY server.py .

# Install any Python dependencies if needed
RUN pip install --no-cache-dir aiofiles

EXPOSE 8080

CMD ["python", "server.py"]