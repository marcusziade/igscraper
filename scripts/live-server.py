#!/usr/bin/env python3
"""
Live reload server for IGScraper documentation.
Automatically refreshes the browser when files change.
"""

import os
import sys
import time
import threading
import webbrowser
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
import socketserver
import signal

# Configuration
PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8888
WATCH_EXTENSIONS = {'.html', '.css', '.js', '.json', '.md'}
CHECK_INTERVAL = 0.5  # seconds

# Colors for terminal output
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
RED = '\033[0;31m'
BLUE = '\033[0;34m'
NC = '\033[0m'  # No Color

# Track file modifications
file_mtimes = {}
should_reload = False
server = None

class LiveReloadHandler(SimpleHTTPRequestHandler):
    """HTTP handler with live reload injection"""
    
    def end_headers(self):
        # Add CORS headers for live reload
        self.send_header('Cache-Control', 'no-store, no-cache, must-revalidate')
        self.send_header('Access-Control-Allow-Origin', '*')
        super().end_headers()
    
    def do_GET(self):
        if self.path == '/__live_reload_check__':
            # Endpoint for checking if reload is needed
            global should_reload
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(f'{{"reload": {str(should_reload).lower()}}}'.encode())
            if should_reload:
                should_reload = False
        else:
            # Inject live reload script into HTML files
            if self.path == '/' or self.path.endswith('.html'):
                # Save original path
                original_path = self.path
                
                # Get the file content
                if self.path == '/':
                    self.path = '/index.html'
                
                file_path = self.translate_path(self.path)
                
                try:
                    with open(file_path, 'rb') as f:
                        content = f.read()
                    
                    # Inject live reload script before closing body tag
                    if b'</body>' in content:
                        reload_script = b'''
<script>
(function() {
    let reloadInterval = setInterval(async () => {
        try {
            const response = await fetch('/__live_reload_check__');
            const data = await response.json();
            if (data.reload) {
                console.log('Reloading page...');
                location.reload();
            }
        } catch (e) {
            // Server might be restarting
        }
    }, 500);
})();
</script>
</body>'''
                        content = content.replace(b'</body>', reload_script)
                    
                    # Send the modified content
                    self.send_response(200)
                    self.send_header('Content-Type', 'text/html')
                    self.send_header('Content-Length', str(len(content)))
                    self.end_headers()
                    self.wfile.write(content)
                    
                    # Restore original path
                    self.path = original_path
                except Exception as e:
                    print(f"{RED}Error serving {self.path}: {e}{NC}")
                    super().do_GET()
            else:
                # Serve other files normally
                super().do_GET()
    
    def log_message(self, format, *args):
        # Custom log format with colors
        if args[1] == '200':
            sys.stderr.write(f"{GREEN}[{self.log_date_time_string()}] {format % args}{NC}\n")
        elif args[1] == '404':
            sys.stderr.write(f"{RED}[{self.log_date_time_string()}] {format % args}{NC}\n")
        else:
            sys.stderr.write(f"[{self.log_date_time_string()}] {format % args}\n")

def watch_files(directory):
    """Watch for file changes in the directory"""
    global file_mtimes, should_reload
    
    print(f"{BLUE}Watching for changes in: {directory}{NC}")
    
    while True:
        try:
            for root, dirs, files in os.walk(directory):
                # Skip hidden directories and common build directories
                dirs[:] = [d for d in dirs if not d.startswith('.') and d not in ['node_modules', '__pycache__']]
                
                for file in files:
                    if any(file.endswith(ext) for ext in WATCH_EXTENSIONS):
                        filepath = os.path.join(root, file)
                        try:
                            mtime = os.path.getmtime(filepath)
                            
                            if filepath in file_mtimes:
                                if mtime > file_mtimes[filepath]:
                                    print(f"{YELLOW}File changed: {os.path.relpath(filepath, directory)}{NC}")
                                    should_reload = True
                            
                            file_mtimes[filepath] = mtime
                        except OSError:
                            # File might have been deleted
                            pass
            
            time.sleep(CHECK_INTERVAL)
        except Exception as e:
            print(f"{RED}Watch error: {e}{NC}")
            time.sleep(1)

def get_local_ip():
    """Get local IP address for network access"""
    import socket
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except:
        return "localhost"

def signal_handler(signum, frame):
    """Handle Ctrl+C gracefully"""
    print(f"\n{YELLOW}Shutting down server...{NC}")
    if server:
        server.shutdown()
    sys.exit(0)

def main():
    global server
    
    # Set up signal handler
    signal.signal(signal.SIGINT, signal_handler)
    
    # Check if docs directory exists
    docs_dir = Path("docs")
    if not docs_dir.exists():
        print(f"{RED}Error: docs directory not found{NC}")
        print("Please run this script from the project root")
        sys.exit(1)
    
    # Change to docs directory
    os.chdir(docs_dir)
    
    # Get local IP
    local_ip = get_local_ip()
    
    print(f"{GREEN}Starting IGScraper documentation preview with live reload...{NC}")
    print(f"{GREEN}Local access:   http://localhost:{PORT}{NC}")
    print(f"{GREEN}Network access: http://{local_ip}:{PORT}{NC}")
    print(f"{YELLOW}Press Ctrl+C to stop the server{NC}")
    print()
    
    # Start file watcher in a separate thread
    watcher = threading.Thread(target=watch_files, args=(os.getcwd(),), daemon=True)
    watcher.start()
    
    # Open browser after a short delay
    def open_browser_delayed():
        time.sleep(1.5)
        webbrowser.open(f'http://localhost:{PORT}')
    
    browser_thread = threading.Thread(target=open_browser_delayed, daemon=True)
    browser_thread.start()
    
    # Start the server
    Handler = LiveReloadHandler
    
    try:
        with socketserver.TCPServer(("0.0.0.0", PORT), Handler) as httpd:
            server = httpd
            httpd.allow_reuse_address = True
            print(f"{BLUE}Server running on port {PORT}...{NC}\n")
            httpd.serve_forever()
    except OSError as e:
        if e.errno == 98:  # Address already in use
            print(f"{RED}Port {PORT} is already in use!{NC}")
            print(f"Try a different port: {sys.argv[0]} 3000")
        else:
            print(f"{RED}Server error: {e}{NC}")
        sys.exit(1)

if __name__ == "__main__":
    main()