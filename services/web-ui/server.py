#!/usr/bin/env python3
"""
Simple HTTP server to serve the QuantumLayer Platform Web UI
"""
import http.server
import socketserver
import os
from http.server import SimpleHTTPRequestHandler

class CORSRequestHandler(SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        super().end_headers()

    def do_OPTIONS(self):
        self.send_response(200)
        self.end_headers()

def serve():
    PORT = 8080
    Handler = CORSRequestHandler
    
    # Change to the directory containing index.html
    web_dir = os.path.dirname(os.path.abspath(__file__))
    os.chdir(web_dir)
    
    with socketserver.TCPServer(("", PORT), Handler) as httpd:
        print(f"🚀 QuantumLayer Platform Web UI")
        print(f"📍 Server running at http://localhost:{PORT}")
        print(f"🌐 Open http://localhost:{PORT} in your browser")
        print(f"Press Ctrl+C to stop")
        httpd.serve_forever()

if __name__ == "__main__":
    serve()