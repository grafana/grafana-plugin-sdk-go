"""Deterministic upstream fixture; Grafana still runs its real Prometheus plugin."""

import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        path = urlparse(self.path).path
        if path == "/api/v1/status/buildinfo":
            data = {"version": "3.2.1"}
        elif path == "/api/v1/query":
            data = {"resultType": "vector", "result": [
                {"metric": {"__name__": "fixture_bytes", "host": "a"}, "value": [1767323045, "10"]},
                {"metric": {"__name__": "fixture_bytes", "host": "b"}, "value": [1767323045, "20"]},
            ]}
        elif path == "/api/v1/query_range":
            data = {"resultType": "matrix", "result": [
                {"metric": {"__name__": "fixture_bytes", "host": "a"}, "values": [[1767323045, "10"], [1767323046, "11"]]},
                {"metric": {"__name__": "fixture_bytes", "host": "b"}, "values": [[1767323045, "20"], [1767323046, "21"]]},
            ]}
        else:
            self.send_error(404)
            return
        body = json.dumps({"status": "success", "data": data}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


HTTPServer(("0.0.0.0", 9090), Handler).serve_forever()
