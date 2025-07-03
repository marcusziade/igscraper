#!/usr/bin/env python3
"""
Live-reloading server for IGScraper documentation
Serves on port 8888 and is accessible from the entire network
"""

import os
import sys
import time
import signal
import socket
import subprocess
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
import threading
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler

# Configuration
PORT = 8888
DOCS_DIR = Path(__file__).parent.parent / "docs"

class LiveReloadHandler(SimpleHTTPRequestHandler):
    """HTTP handler with live reload injection"""
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(DOCS_DIR), **kwargs)
    
    def end_headers(self):
        # Add CORS headers for local development
        self.send_header('Cache-Control', 'no-store, no-cache, must-revalidate')
        self.send_header('Access-Control-Allow-Origin', '*')
        super().end_headers()
    
    def do_GET(self):
        """Inject live reload script into HTML files"""
        if self.path == '/' or self.path.endswith('.html'):
            # Serve the file with injected reload script
            file_path = DOCS_DIR / (self.path.lstrip('/') or 'index.html')
            if file_path.exists() and file_path.suffix == '.html':
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                # Inject live reload script before closing body tag
                reload_script = """
<script>
(function() {
    let lastModified = null;
    
    async function checkForChanges() {
        try {
            const response = await fetch('/_check_reload');
            const data = await response.json();
            
            if (lastModified && data.modified !== lastModified) {
                console.log('Changes detected, reloading...');
                window.location.reload();
            }
            lastModified = data.modified;
        } catch (e) {
            // Server might be restarting
        }
    }
    
    // Check every 500ms
    setInterval(checkForChanges, 500);
    
    // Also listen for focus to reload immediately when switching back
    window.addEventListener('focus', checkForChanges);
})();
</script>
"""
                content = content.replace('</body>', reload_script + '</body>')
                
                # Send response
                self.send_response(200)
                self.send_header('Content-Type', 'text/html; charset=utf-8')
                self.send_header('Content-Length', str(len(content.encode('utf-8'))))
                self.end_headers()
                self.wfile.write(content.encode('utf-8'))
                return
        
        if self.path == '/_check_reload':
            # Return last modification time for polling
            last_modified = get_last_modified_time()
            response = f'{{"modified": {last_modified}}}'
            
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(response)))
            self.end_headers()
            self.wfile.write(response.encode('utf-8'))
            return
        
        # Default handler for other files
        super().do_GET()
    
    def log_message(self, format, *args):
        """Suppress request logging for cleaner output"""
        if '/_check_reload' not in args[0]:
            super().log_message(format, *args)

def get_last_modified_time():
    """Get the most recent modification time in the docs directory"""
    latest = 0
    for root, dirs, files in os.walk(DOCS_DIR):
        # Skip hidden directories
        dirs[:] = [d for d in dirs if not d.startswith('.')]
        
        for file in files:
            if not file.startswith('.'):
                file_path = Path(root) / file
                mtime = file_path.stat().st_mtime
                latest = max(latest, mtime)
    return latest

def get_local_ip():
    """Get the local IP address"""
    try:
        # Create a socket to determine the local IP
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except:
        return "localhost"

def kill_existing_server():
    """Kill any existing process on port 8888"""
    try:
        # Try to find process using the port
        if sys.platform == "darwin":  # macOS
            cmd = f"lsof -ti:{PORT}"
        else:  # Linux
            cmd = f"fuser -k {PORT}/tcp 2>/dev/null"
        
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        if result.stdout.strip():
            print(f"🔄 Killed existing process on port {PORT}")
            time.sleep(1)  # Give it time to release the port
    except:
        pass

def main():
    """Start the live-reloading server"""
    # Kill any existing server
    kill_existing_server()
    
    # Change to docs directory
    os.chdir(DOCS_DIR)
    
    # Get network IP
    local_ip = get_local_ip()
    
    # Create server
    server = HTTPServer(('0.0.0.0', PORT), LiveReloadHandler)
    
    print(f"""
🚀 IGScraper Docs Server Started!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Local:    http://localhost:{PORT}
📍 Network:  http://{local_ip}:{PORT}

✨ Features:
  • Live reload on file changes
  • Network accessible
  • Auto-refresh every 500ms

Press Ctrl+C to stop
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
""")
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n\n👋 Server stopped")
        server.shutdown()

if __name__ == '__main__':
    # Check if docs directory exists
    if not DOCS_DIR.exists():
        print(f"❌ Error: Docs directory not found at {DOCS_DIR}")
        sys.exit(1)
    
    # Check for required dependencies
    try:
        import watchdog
    except ImportError:
        print("📦 Installing required dependency: watchdog")
        subprocess.run([sys.executable, "-m", "pip", "install", "watchdog"], check=True)
        from watchdog.observers import Observer
        from watchdog.events import FileSystemEventHandler
    
    main()