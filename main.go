package main

import (
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/hashicorp/yamux"
)

//go:embed Tatbeeblink-logo.png
var iconData []byte

const (
	Version     = "1.0.0"
	RelayServer = "link.tatbeeb.sa:8443"
	WebPort     = "8765"
)

type App struct {
	connected     bool
	localPort     string
	shareablePort string
	shareableLink string
	relayConn     net.Conn
	yamuxSession  *yamux.Session
	statusChannel chan StatusUpdate
	tunnelActive  bool
	tunnelMutex   sync.RWMutex

	// Tray menu items
	mStatus *systray.MenuItem
	mOpen   *systray.MenuItem
	mQuit   *systray.MenuItem
}

type StatusUpdate struct {
	Connected     bool   `json:"connected"`
	Status        string `json:"status"`
	ShareableLink string `json:"shareableLink"`
	LocalPort     string `json:"localPort"`
	Error         string `json:"error"`
}

type ConnectRequest struct {
	LocalPort string `json:"localPort"`
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("Tatbeeb Link")
	systray.SetTooltip("Tatbeeb Link - Secure Port Tunneling")

	app := &App{
		statusChannel: make(chan StatusUpdate, 10),
		localPort:     "9999", // Default port
	}

	// Create menu items
	app.mStatus = systray.AddMenuItem("Status: Disconnected", "Connection status")
	app.mStatus.Disable()

	systray.AddSeparator()

	app.mOpen = systray.AddMenuItem("Open Dashboard", "Open web interface")

	systray.AddSeparator()

	app.mQuit = systray.AddMenuItem("Exit", "Quit Tatbeeb Link")

	// Start HTTP server in background
	go app.startWebServer()

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-app.mOpen.ClickedCh:
				openBrowser("http://localhost:" + WebPort)
			case <-app.mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Update status periodically
	go app.updateTrayStatus()
}

func onExit() {
	log.Println("Tatbeeb Link shutting down...")
}

func (a *App) startWebServer() {
	http.HandleFunc("/", a.handleIndex)
	http.HandleFunc("/api/status", a.handleStatus)
	http.HandleFunc("/api/connect", a.handleConnect)
	http.HandleFunc("/api/disconnect", a.handleDisconnect)

	addr := "localhost:" + WebPort
	url := "http://" + addr

	log.Printf("🚀 Tatbeeb Link v%s starting...", Version)
	log.Printf("🌐 Web interface available at %s", url)
	log.Printf("📊 System tray icon active")

	go func() {
		time.Sleep(1 * time.Second)
		openBrowser(url)
	}()

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func (a *App) updateTrayStatus() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if a.connected && a.shareableLink != "" {
			a.mStatus.SetTitle(fmt.Sprintf("Connected: %s", a.shareableLink))
		} else {
			a.mStatus.SetTitle("Status: Disconnected")
		}
	}
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tatbeeb Link</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #2563eb 0%, #1e40af 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 500px;
            width: 100%;
            padding: 40px;
        }
        h1 {
            color: #000000;
            font-size: 38px;
            margin-bottom: 10px;
            text-align: center;
        }
        .subtitle {
            color: #666;
            text-align: center;
            margin-bottom: 30px;
        }
        .status {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 10px;
            padding: 20px;
            background: #f7fafc;
            border-radius: 10px;
            margin-bottom: 30px;
        }
        .status-dot {
            width: 16px;
            height: 16px;
            border-radius: 50%;
            background: #ef4444;
            animation: pulse 2s infinite;
        }
        .status-dot.connected {
            background: #22c55e;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
        .status-text {
            font-weight: 600;
            color: #1f2937;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            color: #374151;
            font-weight: 500;
            font-size: 14px;
        }
        input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e5e7eb;
            border-radius: 8px;
            font-size: 14px;
            transition: all 0.3s;
        }
        input:focus {
            outline: none;
            border-color: #2563eb;
            box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
        }
        .button {
            width: 100%;
            padding: 14px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
            margin-bottom: 10px;
        }
        .button-primary {
            background: #2563eb;
            color: white;
        }
        .button-primary:hover {
            background: #1d4ed8;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(37, 99, 235, 0.4);
        }
        .button-danger {
            background: #ef4444;
            color: white;
        }
        .button-danger:hover {
            background: #dc2626;
        }
        .button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .shareable-box {
            background: #ecfdf5;
            border: 2px solid #10b981;
            border-radius: 10px;
            padding: 20px;
            margin: 20px 0;
            display: none;
        }
        .shareable-box.show {
            display: block;
        }
        .shareable-label {
            font-size: 12px;
            color: #047857;
            font-weight: 600;
            margin-bottom: 8px;
        }
        .shareable-link {
            font-size: 18px;
            font-weight: 700;
            color: #065f46;
            margin-bottom: 12px;
            word-break: break-all;
        }
        .info-box {
            background: #eff6ff;
            border-left: 4px solid #2563eb;
            padding: 16px;
            margin: 20px 0;
            border-radius: 8px;
        }
        .info-box ul {
            list-style: none;
            padding-left: 0;
        }
        .info-box li {
            padding: 4px 0;
            color: #1e40af;
            font-size: 14px;
        }
        .info-box li:before {
            content: "✓ ";
            color: #2563eb;
            font-weight: bold;
            margin-right: 8px;
        }
        .error {
            background: #fee2e2;
            border-left: 4px solid #ef4444;
            padding: 12px;
            margin: 10px 0;
            border-radius: 8px;
            color: #991b1b;
            font-size: 14px;
            display: none;
        }
        .error.show {
            display: block;
        }
        .setup-form {
            display: block;
        }
        .setup-form.hidden {
            display: none;
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            color: #9ca3af;
            font-size: 12px;
        }
        .tray-notice {
            background: #fef3c7;
            border-left: 4px solid #f59e0b;
            padding: 12px;
            margin: 20px 0;
            border-radius: 8px;
            font-size: 13px;
            color: #92400e;
        }
        .button-secondary {
            background: #f3f4f6;
            color: #374151;
        }
        .button-secondary:hover {
            background: #e5e7eb;
        }
    </style>
</head>
<body>
    <div class="container">
        <div style="text-align: center; margin-bottom: 20px;">
            <img src="https://i.postimg.cc/rwZ7Sqpd/Tatbeeblink-logo.png" alt="Tatbeeb Link" style="width: 160px; height: 160px; margin: 0 auto 10px;">
        </div>
        <h1>Tatbeeb Link</h1>
        <p class="subtitle">Connect your Database to Tatbeeb HIS</p>
        
        <div class="status">
            <div class="status-dot" id="statusDot"></div>
            <span class="status-text" id="statusText">Disconnected</span>
        </div>

        <div class="error" id="errorBox"></div>

        <div class="shareable-box" id="shareableBox">
            <div class="shareable-label">📋 SHAREABLE LINK</div>
            <div class="shareable-link" id="shareableLink"></div>
            <div class="text-sm text-green-800 mb-3">
                Forward this address to anyone. Connections to this address will be tunneled to your local port <span id="displayLocalPort">9999</span>.
            </div>
            <button class="button button-secondary" onclick="copyLink()">📋 Copy Link</button>
        </div>

        <div class="setup-form" id="setupForm">
            <button class="button button-primary" onclick="connect()" id="connectBtn">🚀 Start Connection</button>

            <div class="advanced-settings" style="margin-top: 20px;">
                <button class="button button-secondary" onclick="toggleAdvanced()" id="advancedBtn">⚙️ Advanced Settings</button>
                <div id="advancedPanel" style="display: none; margin-top: 15px;">
                    <div class="form-group">
                        <label>Local Port to Tunnel</label>
                        <input type="number" id="localPort" value="9999" placeholder="9999" min="1" max="65535">
                    </div>
                </div>
            </div>
        </div>

        <div class="setup-form hidden" id="connectedForm">
            <div class="info-box">
                <ul>
                    <li>Tunnel active</li>
                    <li>Port forwarding enabled</li>
                    <li>Share link with anyone</li>
                </ul>
            </div>
            <button class="button button-danger" onclick="disconnect()">⏹️ Stop Tunnel</button>
        </div>

        <div class="footer">
            © 2025 Tatbeeb Healthcare Technology<br>
            Version 1.0.0 • Running in system tray
        </div>
    </div>

    <script>
        let pollingInterval;

        function showError(message) {
            const errorBox = document.getElementById('errorBox');
            errorBox.textContent = message;
            errorBox.classList.add('show');
            setTimeout(() => errorBox.classList.remove('show'), 5000);
        }

        async function connect() {
            const localPort = document.getElementById('localPort').value;

            if (!localPort || localPort < 1 || localPort > 65535) {
                showError('Please enter a valid port number (1-65535)');
                return;
            }

            document.getElementById('connectBtn').disabled = true;
            document.getElementById('statusText').textContent = 'Connecting to relay...';

            try {
                const response = await fetch('/api/connect', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ localPort })
                });

                const result = await response.json();

                if (result.success) {
                    document.getElementById('setupForm').classList.add('hidden');
                    document.getElementById('connectedForm').classList.remove('hidden');
                    startPolling();
                } else {
                    showError('Connection failed: ' + result.error);
                    document.getElementById('connectBtn').disabled = false;
                }
            } catch (error) {
                showError('Connect failed: ' + error.message);
                document.getElementById('connectBtn').disabled = false;
            }
        }

        async function disconnect() {
            try {
                await fetch('/api/disconnect', { method: 'POST' });
                stopPolling();
                document.getElementById('setupForm').classList.remove('hidden');
                document.getElementById('connectedForm').classList.add('hidden');
                document.getElementById('shareableBox').classList.remove('show');
                document.getElementById('connectBtn').disabled = false;
            } catch (error) {
                showError('Disconnect failed: ' + error.message);
            }
        }

        async function updateStatus() {
            try {
                const response = await fetch('/api/status');
                const status = await response.json();

                const statusDot = document.getElementById('statusDot');
                const statusText = document.getElementById('statusText');
                const shareableBox = document.getElementById('shareableBox');
                const shareableLink = document.getElementById('shareableLink');
                const displayLocalPort = document.getElementById('displayLocalPort');

                if (status.connected) {
                    statusDot.classList.add('connected');
                    statusText.textContent = 'Tunnel Active';
                    shareableLink.textContent = status.shareableLink;
                    displayLocalPort.textContent = status.localPort;
                    shareableBox.classList.add('show');
                } else {
                    statusDot.classList.remove('connected');
                    statusText.textContent = status.status || 'Disconnected';
                    shareableBox.classList.remove('show');
                }

                if (status.error) {
                    showError(status.error);
                }
            } catch (error) {
                console.error('Failed to update status:', error);
            }
        }

        function startPolling() {
            updateStatus();
            pollingInterval = setInterval(updateStatus, 2000);
        }

        function stopPolling() {
            if (pollingInterval) {
                clearInterval(pollingInterval);
            }
        }

        function copyLink() {
            const link = document.getElementById('shareableLink').textContent;
            navigator.clipboard.writeText(link).then(() => {
                const btn = event.target;
                const originalText = btn.textContent;
                btn.textContent = '✅ Copied!';
                setTimeout(() => btn.textContent = originalText, 2000);
            });
        }

        function toggleAdvanced() {
            const panel = document.getElementById('advancedPanel');
            const btn = document.getElementById('advancedBtn');
            if (panel.style.display === 'none') {
                panel.style.display = 'block';
                btn.textContent = '⚙️ Hide Advanced Settings';
            } else {
                panel.style.display = 'none';
                btn.textContent = '⚙️ Advanced Settings';
            }
        }

        startPolling();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := StatusUpdate{
		Connected:     a.connected,
		Status:        "Disconnected",
		ShareableLink: a.shareableLink,
		LocalPort:     a.localPort,
	}

	if a.connected {
		status.Status = "Tunnel Active"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (a *App) handleConnect(w http.ResponseWriter, r *http.Request) {
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// Save local port
	a.localPort = req.LocalPort

	// Start tunnel to relay
	shareablePort, err := a.startTunnel()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Tunnel failed: " + err.Error(),
		})
		return
	}

	a.shareablePort = shareablePort
	a.shareableLink = fmt.Sprintf("link.tatbeeb.sa:%s", shareablePort)
	a.connected = true

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"shareableLink": a.shareableLink,
		"localPort":     a.localPort,
	})
}

func (a *App) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	a.closeTunnel()
	a.connected = false
	a.shareablePort = ""
	a.shareableLink = ""

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (a *App) startTunnel() (string, error) {
	log.Printf("🚀 Starting tunnel to %s...", RelayServer)

	tlsConfig := &tls.Config{
		ServerName: "link.tatbeeb.sa",
	}

	log.Printf("🔐 Connecting to relay with TLS...")
	conn, err := tls.Dial("tcp", RelayServer, tlsConfig)
	if err != nil {
		log.Printf("❌ Failed to connect to relay: %v", err)
		return "", fmt.Errorf("failed to connect to relay: %w", err)
	}
	log.Printf("✅ TLS connection established")

	a.relayConn = conn

	// Send REGISTER command
	registerMsg := "REGISTER\n"
	log.Printf("📤 Sending REGISTER command...")
	_, err = conn.Write([]byte(registerMsg))
	if err != nil {
		log.Printf("❌ Failed to send register: %v", err)
		conn.Close()
		return "", fmt.Errorf("failed to send register: %w", err)
	}
	log.Printf("✅ REGISTER command sent")

	// Read response byte-by-byte to avoid buffering issues with yamux
	log.Printf("📖 Reading response from relay...")
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var response strings.Builder
	buf := make([]byte, 1)
	bytesRead := 0
	for {
		_, err := conn.Read(buf)
		if err != nil {
			log.Printf("❌ Failed to read response after %d bytes: %v", bytesRead, err)
			conn.Close()
			return "", fmt.Errorf("failed to read response: %w", err)
		}
		bytesRead++
		if buf[0] == '\n' {
			break
		}
		response.WriteByte(buf[0])
	}
	conn.SetReadDeadline(time.Time{})

	responseStr := strings.TrimSpace(response.String())
	log.Printf("📥 Received response: '%s' (%d bytes)", responseStr, bytesRead)

	parts := strings.Split(responseStr, " ")
	if len(parts) < 2 || parts[0] != "OK" {
		log.Printf("❌ Unexpected response format: %s", responseStr)
		conn.Close()
		return "", fmt.Errorf("unexpected response: %s", responseStr)
	}

	portParts := strings.Split(parts[1], ":")
	if len(portParts) != 2 || portParts[0] != "port" {
		log.Printf("❌ Invalid port format: %s", parts[1])
		conn.Close()
		return "", fmt.Errorf("invalid port format: %s", parts[1])
	}

	shareablePort := portParts[1]
	log.Printf("✅ Assigned port: %s", shareablePort)

	// Create yamux client session for multiplexing
	log.Printf("🔀 Creating yamux client session...")
	session, err := yamux.Client(conn, nil)
	if err != nil {
		log.Printf("❌ Failed to create yamux session: %v", err)
		conn.Close()
		return "", fmt.Errorf("failed to create yamux session: %w", err)
	}
	a.yamuxSession = session
	log.Printf("✅ Yamux session created")

	// Start accepting incoming streams (client connections)
	log.Printf("🎧 Starting to accept streams...")
	go a.acceptStreams()

	log.Printf("✅ Tunnel ready: localhost:%s -> link.tatbeeb.sa:%s", a.localPort, shareablePort)

	return shareablePort, nil
}

func (a *App) acceptStreams() {
	log.Printf("🎧 Ready to accept streams from relay...")
	streamCount := 0

	for {
		// Accept incoming streams from relay (each stream = one client connection)
		log.Printf("⏳ Waiting for next stream...")
		stream, err := a.yamuxSession.AcceptStream()
		if err != nil {
			log.Printf("❌ Session closed: %v", err)
			a.connected = false
			return
		}

		streamCount++
		log.Printf("🔗 Stream #%d accepted from relay", streamCount)

		// Handle each stream in a goroutine
		go a.handleStream(stream, streamCount)
	}
}

func (a *App) handleStream(stream net.Conn, streamNum int) {
	defer stream.Close()

	log.Printf("🔗 [Stream#%d] New stream from relay, connecting to localhost:%s", streamNum, a.localPort)

	// Connect to local port
	localAddr := fmt.Sprintf("localhost:%s", a.localPort)
	log.Printf("📡 [Stream#%d] Dialing %s...", streamNum, localAddr)
	localConn, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		log.Printf("❌ [Stream#%d] Failed to connect to local port %s: %v", streamNum, a.localPort, err)
		return
	}
	defer localConn.Close()

	log.Printf("✅ [Stream#%d] Connected to local port, starting data forwarding...", streamNum)

	// Forward data bidirectionally
	done := make(chan bool, 2)

	// Stream -> Local
	go func() {
		n, err := io.Copy(localConn, stream)
		if err != nil {
			log.Printf("⚠️ [Stream#%d] Relay->Local error: %v", streamNum, err)
		}
		log.Printf("📥 [Stream#%d] Relay->Local: %d bytes", streamNum, n)
		done <- true
	}()

	// Local -> Stream
	go func() {
		n, err := io.Copy(stream, localConn)
		if err != nil {
			log.Printf("⚠️ [Stream#%d] Local->Relay error: %v", streamNum, err)
		}
		log.Printf("📤 [Stream#%d] Local->Relay: %d bytes", streamNum, n)
		done <- true
	}()

	// Wait for either direction to close
	<-done
	log.Printf("🔌 [Stream#%d] Connection closed", streamNum)
}

func (a *App) closeTunnel() error {
	if a.yamuxSession != nil {
		a.yamuxSession.Close()
		a.yamuxSession = nil
	}
	if a.relayConn != nil {
		a.relayConn.Close()
		a.relayConn = nil
	}
	return nil
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func getIcon() []byte {
	// Return the embedded icon data
	if len(iconData) > 0 {
		return iconData
	}

	// Fallback: Try to load from file system
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		return []byte{}
	}

	iconPath := filepath.Join(filepath.Dir(exePath), "Tatbeeblink-logo.png")
	fileData, err := os.ReadFile(iconPath)
	if err != nil {
		// If not found next to executable, try current directory
		fileData, err = os.ReadFile("Tatbeeblink-logo.png")
		if err != nil {
			log.Printf("Failed to load icon: %v", err)
			return []byte{}
		}
	}

	return fileData
}
