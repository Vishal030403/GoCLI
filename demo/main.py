"""Pipeline CLI demo — opens in your default browser."""

import json
import webbrowser
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import parse_qs, urlparse

HOST = "127.0.0.1"
PORT = 8080


def html_page(greet: str = "", count: int = 0) -> bytes:
    greet_block = (
        f'<p class="greet">{greet}</p>'
        if greet
        else '<p class="greet muted">Enter your name and click Say hello.</p>'
    )
    body = f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Pipeline CLI Demo</title>
  <style>
    * {{ box-sizing: border-box; }}
    body {{
      font-family: "Segoe UI", system-ui, sans-serif;
      margin: 0; min-height: 100vh;
      background: linear-gradient(145deg, #0f172a 0%, #1e3a5f 50%, #0f172a 100%);
      color: #e2e8f0; display: flex; align-items: center; justify-content: center;
      padding: 24px;
    }}
    .card {{
      background: rgba(15, 23, 42, 0.85);
      border: 1px solid rgba(148, 163, 184, 0.2);
      border-radius: 16px; padding: 32px; max-width: 420px; width: 100%;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    }}
    h1 {{ margin: 0 0 8px; font-size: 1.5rem; color: #f8fafc; }}
    .sub {{ color: #94a3b8; font-size: 0.9rem; margin-bottom: 24px; }}
    section {{ margin-bottom: 20px; }}
    section h2 {{ font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.08em;
      color: #64748b; margin: 0 0 10px; }}
    input[type="text"] {{
      width: 100%; padding: 10px 12px; border-radius: 8px;
      border: 1px solid #334155; background: #0f172a; color: #f1f5f9;
      font-size: 1rem;
    }}
    button {{
      margin-top: 10px; padding: 10px 18px; border: none; border-radius: 8px;
      background: #38bdf8; color: #0f172a; font-weight: 600; cursor: pointer;
      font-size: 0.95rem;
    }}
    button:hover {{ background: #7dd3fc; }}
    .counter {{ display: flex; align-items: center; gap: 16px; }}
    .counter button {{ margin: 0; width: 44px; height: 44px; font-size: 1.25rem; }}
    .count {{ font-size: 2rem; font-weight: 700; min-width: 3ch; text-align: center; }}
    .greet {{ margin: 12px 0 0; color: #a5f3fc; }}
    .greet.muted {{ color: #64748b; }}
    .status {{ margin-top: 20px; font-size: 0.85rem; color: #4ade80; }}
    a {{ color: #38bdf8; }}
  </style>
</head>
<body>
  <div class="card">
    <h1>Pipeline CLI Demo</h1>
    <p class="sub">Running at <a href="http://{HOST}:{PORT}/">http://{HOST}:{PORT}/</a></p>

    <section>
      <h2>Say hello</h2>
      <form method="get" action="/">
        <input type="text" name="name" placeholder="Your name" autofocus>
        <button type="submit">Say hello</button>
      </form>
      {greet_block}
    </section>

    <section>
      <h2>Counter</h2>
      <div class="counter">
        <a href="/?count={count - 1}"><button type="button">−</button></a>
        <span class="count">{count}</span>
        <a href="/?count={count + 1}"><button type="button">+</button></a>
      </div>
    </section>

    <p class="status">Status: Ready</p>
  </div>
</body>
</html>"""
    return body.encode("utf-8")


class DemoHandler(BaseHTTPRequestHandler):
    count = 0

    def log_message(self, fmt, *args):
        print(f"[demo] {self.address_string()} - {fmt % args}")

    def do_GET(self):
        parsed = urlparse(self.path)
        if parsed.path not in ("/", "/health"):
            self.send_error(404)
            return

        if parsed.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"status": "ok"}).encode())
            return

        qs = parse_qs(parsed.query)
        greet = ""
        if "name" in qs and qs["name"][0].strip():
            greet = f"Hello, {qs['name'][0].strip()}! Welcome to the Pipeline CLI demo."

        if "count" in qs:
            try:
                DemoHandler.count = int(qs["count"][0])
            except ValueError:
                pass

        body = html_page(greet=greet, count=DemoHandler.count)
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.end_headers()
        self.wfile.write(body)


def main():
    url = f"http://{HOST}:{PORT}/"
    server = HTTPServer((HOST, PORT), DemoHandler)
    print(f"Pipeline CLI demo: {url}")
    print("Press Ctrl+C to stop.")
    webbrowser.open(url)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")
        server.server_close()


if __name__ == "__main__":
    main()
