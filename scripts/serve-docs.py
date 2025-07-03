#!/usr/bin/env python3
"""
Simple HTTP server for IGScraper documentation
No dependencies required - uses only Python standard library
"""

import os
import sys
import time
import signal
import socket
import subprocess
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path

# Configuration
PORT = 8888
DOCS_DIR = Path(__file__).parent.parent / "docs"

# Global variable to track file changes
last_check_time = time.time()

class ReloadHandler(SimpleHTTPRequestHandler):
    """HTTP handler with basic auto-reload functionality"""
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(DOCS_DIR), **kwargs)
    
    def end_headers(self):
        # Disable caching
        self.send_header('Cache-Control', 'no-store, no-cache, must-revalidate')
        self.send_header('Expires', '0')
        super().end_headers()
    
    def do_GET(self):
        """Inject reload script into HTML files"""
        if self.path == '/' or self.path.endswith('.html'):
            file_path = DOCS_DIR / (self.path.lstrip('/') or 'index.html')
            if file_path.exists() and file_path.suffix == '.html':
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                # Simple reload script that checks every second
                reload_script = """
<script>
// Auto-reload functionality
(function() {
    let lastCheck = Date.now();
    
    setInterval(function() {
        fetch(window.location.href, { method: 'HEAD' })
            .then(response => {
                const modified = response.headers.get('Last-Modified');
                if (modified && lastCheck && Date.parse(modified) > lastCheck) {
                    location.reload();
                }
            }).catch(() => {});
    }, 1000);
})();
</script>
"""
                content = content.replace('</body>', reload_script + '</body>')
                
                # Send response
                self.send_response(200)
                self.send_header('Content-Type', 'text/html; charset=utf-8')
                self.send_header('Last-Modified', time.strftime('%a, %d %b %Y %H:%M:%S GMT', time.gmtime()))
                self.end_headers()
                self.wfile.write(content.encode('utf-8'))
                return
        
        # Default handler for other files
        super().do_GET()
    
    def log_message(self, format, *args):
        """Minimal logging"""
        # Only log non-GET requests and errors
        if "GET" not in args[0] or args[1] != '200':
            sys.stdout.write(f"[{self.log_date_time_string()}] {format % args}\n")

def get_local_ip():
    """Get the local IP address"""
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except:
        return "127.0.0.1"

def kill_port(port):
    """Kill process on the specified port"""
    try:
        if sys.platform == "darwin":  # macOS
            os.system(f"lsof -ti:{port} | xargs kill -9 2>/dev/null")
        else:  # Linux
            os.system(f"fuser -k {port}/tcp 2>/dev/null")
        time.sleep(0.5)
    except:
        pass

def watch_files():
    """Simple file watcher that touches index.html when changes detected"""
    global last_check_time
    while True:
        time.sleep(1)
        try:
            # Check if any file has been modified
            for root, dirs, files in os.walk(DOCS_DIR):
                dirs[:] = [d for d in dirs if not d.startswith('.')]
                for file in files:
                    if not file.startswith('.') and not file.endswith('.swp'):
                        file_path = Path(root) / file
                        mtime = os.path.getmtime(file_path)
                        if mtime > last_check_time:
                            last_check_time = mtime
                            # Touch index.html to trigger reload
                            index_path = DOCS_DIR / 'index.html'
                            if index_path.exists():
                                os.utime(index_path, None)
        except:
            pass

def main():
    """Start the server"""
    # Kill existing process on port
    print(f"🔄 Checking port {PORT}...")
    kill_port(PORT)
    
    # Verify docs directory
    if not DOCS_DIR.exists():
        print(f"❌ Error: Docs directory not found at {DOCS_DIR}")
        sys.exit(1)
    
    # Get network info
    local_ip = get_local_ip()
    
    # Start file watcher in background
    watcher = threading.Thread(target=watch_files, daemon=True)
    watcher.start()
    
    # Create and start server
    os.chdir(DOCS_DIR)
    server = HTTPServer(('0.0.0.0', PORT), ReloadHandler)
    
    print(f"""
╭─────────────────────────────────────╮
│   IGScraper Documentation Server    │
╰─────────────────────────────────────╯

  Local:    http://localhost:{PORT}
  Network:  http://{local_ip}:{PORT}

  • Auto-reloads on file changes
  • No extra dependencies required
  • Press Ctrl+C to stop

""")
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n✋ Server stopped")
        server.shutdown()

if __name__ == '__main__':
    main()