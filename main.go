package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"image/color"
	"log"
	"net"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	_ "github.com/denisenkom/go-mssqldb"
)

const (
	Version     = "1.0.0"
	RelayServer = "link.tatbeeb.sa:8443"
)

type TatbeebLinkApp struct {
	app    fyne.App
	window fyne.Window

	// UI elements
	statusLabel *widget.Label
	statusIcon  *canvas.Circle
	connectBtn  *widget.Button
	settingsBtn *widget.Button

	// State
	connected bool

	// Connection details
	dbServer   string
	dbPort     string
	dbName     string
	dbUser     string
	dbPassword string

	// Tunnel info
	shareablePort  string
	tunnelListener net.Listener
	relayConn      net.Conn
}

func main() {
	// Create app
	myApp := app.New()
	myWindow := myApp.NewWindow("Tatbeeb Link")
	myWindow.Resize(fyne.NewSize(400, 500))
	myWindow.SetFixedSize(true)

	// Create TatbeebLink app instance
	tlApp := &TatbeebLinkApp{
		app:       myApp,
		window:    myWindow,
		connected: false,
	}

	// Build UI
	content := tlApp.buildUI()
	myWindow.SetContent(content)

	// Show and run
	myWindow.ShowAndRun()
}

func (t *TatbeebLinkApp) buildUI() fyne.CanvasObject {
	// Header with logo
	title := widget.NewLabelWithStyle("Tatbeeb Link", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	title.TextStyle.Bold = true

	subtitle := widget.NewLabel("Secure Database Connection")
	subtitle.Alignment = fyne.TextAlignCenter

	version := widget.NewLabel(fmt.Sprintf("Version %s", Version))
	version.Alignment = fyne.TextAlignCenter
	version.TextStyle.Italic = true

	header := container.NewVBox(
		layout.NewSpacer(),
		title,
		subtitle,
		version,
		layout.NewSpacer(),
	)

	// Status indicator
	t.statusIcon = canvas.NewCircle(color.RGBA{220, 38, 38, 255}) // Red
	t.statusIcon.Resize(fyne.NewSize(20, 20))

	t.statusLabel = widget.NewLabel("Disconnected")
	t.statusLabel.TextStyle.Bold = true

	statusBox := container.NewHBox(
		layout.NewSpacer(),
		container.NewHBox(t.statusIcon, t.statusLabel),
		layout.NewSpacer(),
	)

	// Info cards
	infoText := `• Connect to your local SQL Server
• Get a shareable TCP link instantly
• No firewall configuration needed
• Secure encrypted tunnel`

	infoLabel := widget.NewLabel(infoText)
	infoCard := container.NewPadded(
		container.NewVBox(
			widget.NewLabel("Features:"),
			infoLabel,
		),
	)

	// Connect button
	t.connectBtn = widget.NewButton("Connect", func() {
		t.onConnectClick()
	})
	t.connectBtn.Importance = widget.HighImportance

	// Settings button
	t.settingsBtn = widget.NewButton("Settings", func() {
		t.showSettings()
	})

	// Help button
	helpBtn := widget.NewButton("Help & Documentation", func() {
		t.showHelp()
	})

	buttons := container.NewVBox(
		t.connectBtn,
		t.settingsBtn,
		helpBtn,
	)

	// Footer
	footer := widget.NewLabel("© 2025 Tatbeeb Healthcare Technology")
	footer.Alignment = fyne.TextAlignCenter
	footer.TextStyle.Italic = true

	// Main layout
	content := container.NewBorder(
		header,
		footer,
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			statusBox,
			layout.NewSpacer(),
			infoCard,
			layout.NewSpacer(),
			buttons,
			layout.NewSpacer(),
		),
	)

	return content
}

func (t *TatbeebLinkApp) onConnectClick() {
	if !t.connected {
		// Show setup wizard
		t.showSetupWizard()
	} else {
		// Disconnect
		t.disconnect()
	}
}

func (t *TatbeebLinkApp) showSetupWizard() {
	// Create wizard window
	wizard := t.app.NewWindow("Connect to SQL Server")
	wizard.Resize(fyne.NewSize(500, 450))
	wizard.SetFixedSize(true)

	// Database Configuration
	dbServerEntry := widget.NewEntry()
	dbServerEntry.SetPlaceHolder("localhost")
	dbServerEntry.SetText("localhost")

	dbPortEntry := widget.NewEntry()
	dbPortEntry.SetPlaceHolder("1433")
	dbPortEntry.SetText("1433")

	dbNameEntry := widget.NewEntry()
	dbNameEntry.SetPlaceHolder("YourDatabaseName")

	dbUserEntry := widget.NewEntry()
	dbUserEntry.SetPlaceHolder("sa")

	dbPassEntry := widget.NewPasswordEntry()
	dbPassEntry.SetPlaceHolder("Database password")

	dbForm := container.NewVBox(
		widget.NewLabelWithStyle("SQL Server Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Enter your SQL Server connection details:"),
		widget.NewLabel("Server:"),
		dbServerEntry,
		widget.NewLabel("Port:"),
		dbPortEntry,
		widget.NewLabel("Database:"),
		dbNameEntry,
		widget.NewLabel("Username:"),
		dbUserEntry,
		widget.NewLabel("Password:"),
		dbPassEntry,
	)

	// Progress
	progressLabel := widget.NewLabel("Complete all fields to continue")
	progressBar := widget.NewProgressBar()

	// Buttons
	testBtn := widget.NewButton("Test Connection", func() {
		if dbServerEntry.Text == "" || dbNameEntry.Text == "" || dbUserEntry.Text == "" || dbPassEntry.Text == "" {
			progressLabel.SetText("❌ Please complete all fields")
			return
		}
		progressLabel.SetText("⏳ Testing connection...")
		progressBar.SetValue(0.5)

		// Test actual MSSQL connection
		go func() {
			err := t.testMSSQLConnection(dbServerEntry.Text, dbPortEntry.Text, dbNameEntry.Text, dbUserEntry.Text, dbPassEntry.Text)
			if err != nil {
				progressLabel.SetText(fmt.Sprintf("❌ Connection failed: %v", err))
				progressBar.SetValue(0)
			} else {
				progressLabel.SetText("✅ Connection successful!")
				progressBar.SetValue(1.0)
			}
		}()
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		wizard.Close()
	})

	connectBtn := widget.NewButton("Connect", func() {
		t.processSetup(wizard, dbServerEntry.Text, dbPortEntry.Text, dbNameEntry.Text,
			dbUserEntry.Text, dbPassEntry.Text, progressLabel, progressBar)
	})
	connectBtn.Importance = widget.HighImportance

	buttonBox := container.NewHBox(
		layout.NewSpacer(),
		testBtn,
		cancelBtn,
		connectBtn,
	)

	// Wizard content
	wizardContent := container.NewBorder(
		nil,
		container.NewVBox(
			progressLabel,
			progressBar,
			buttonBox,
		),
		nil,
		nil,
		container.NewScroll(dbForm),
	)

	wizard.SetContent(wizardContent)
	wizard.Show()
}

func (t *TatbeebLinkApp) processSetup(wizard fyne.Window, server, port, dbname, user, pass string, progressLabel *widget.Label, progressBar *widget.ProgressBar) {
	// Validate inputs
	if server == "" || dbname == "" || user == "" || pass == "" {
		progressLabel.SetText("❌ Please complete all database fields")
		return
	}

	// Show progress
	progressLabel.SetText("⏳ Testing database connection...")
	progressBar.SetValue(0.33)

	go func() {
		// Test MSSQL connection
		err := t.testMSSQLConnection(server, port, dbname, user, pass)
		if err != nil {
			progressLabel.SetText(fmt.Sprintf("❌ Connection failed: %v", err))
			progressBar.SetValue(0)
			return
		}

		progressLabel.SetText("⏳ Starting TCP tunnel...")
		progressBar.SetValue(0.66)

		// Store connection details
		t.dbServer = server
		t.dbPort = port
		t.dbName = dbname
		t.dbUser = user
		t.dbPassword = pass

		// Start tunnel
		shareablePort, err := t.startTunnel()
		if err != nil {
			progressLabel.SetText(fmt.Sprintf("❌ Tunnel failed: %v", err))
			progressBar.SetValue(0)
			return
		}

		t.shareablePort = shareablePort

		progressLabel.SetText("✅ Connected!")
		progressBar.SetValue(1.0)

		// Update main window
		t.connected = true
		t.updateConnectionStatus()

		// Show success dialog with shareable link
		shareableLink := fmt.Sprintf("link.tatbeeb.sa:%s", shareablePort)

		successTitle := widget.NewLabelWithStyle("✅ Connection Active!", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		shareableLabel := widget.NewLabel(fmt.Sprintf("Share this connection:\n\n%s", shareableLink))
		shareableLabel.Wrapping = fyne.TextWrapWord
		shareableLabel.Alignment = fyne.TextAlignCenter
		shareableLabel.TextStyle = fyne.TextStyle{Bold: true}

		infoLabel := widget.NewLabel("\nAnyone with this link can connect to your database.\nThe connection will remain active while this app is running.")
		infoLabel.Wrapping = fyne.TextWrapWord
		infoLabel.Alignment = fyne.TextAlignCenter

		copyBtn := widget.NewButton("Copy Link", func() {
			wizard.Clipboard().SetContent(shareableLink)
		})

		okBtn := widget.NewButton("OK", func() {
			wizard.Close()
		})
		okBtn.Importance = widget.HighImportance

		successContent := container.NewVBox(
			layout.NewSpacer(),
			successTitle,
			layout.NewSpacer(),
			shareableLabel,
			infoLabel,
			layout.NewSpacer(),
			container.NewHBox(layout.NewSpacer(), copyBtn, okBtn, layout.NewSpacer()),
			layout.NewSpacer(),
		)

		wizard.SetContent(successContent)
	}()
}

func (t *TatbeebLinkApp) disconnect() {
	// Close the tunnel
	if err := t.closeTunnel(); err != nil {
		log.Printf("Error closing tunnel: %v", err)
	}

	t.connected = false
	t.shareablePort = ""
	t.updateConnectionStatus()

	// Show info
	infoDialog := widget.NewLabel("TCP tunnel closed.\n\nThe shareable link is no longer active.\nYou can reconnect anytime by clicking Connect.")
	infoDialog.Wrapping = fyne.TextWrapWord

	dialog := container.NewVBox(
		infoDialog,
		widget.NewButton("OK", func() {}),
	)

	dlg := widget.NewModalPopUp(dialog, t.window.Canvas())
	dlg.Show()
}

func (t *TatbeebLinkApp) updateConnectionStatus() {
	if t.connected {
		t.statusIcon.FillColor = color.RGBA{34, 197, 94, 255} // Green
		t.statusLabel.SetText("Connected")
		t.connectBtn.SetText("Disconnect")
	} else {
		t.statusIcon.FillColor = color.RGBA{220, 38, 38, 255} // Red
		t.statusLabel.SetText("Disconnected")
		t.connectBtn.SetText("Connect")
	}
	t.statusIcon.Refresh()
}

func (t *TatbeebLinkApp) showSettings() {
	// Settings window
	settings := t.app.NewWindow("Settings")
	settings.Resize(fyne.NewSize(400, 400))

	// Settings options
	autoStart := widget.NewCheck("Start on Windows startup", func(checked bool) {
		// TODO: Implement
	})

	notifications := widget.NewCheck("Show notifications", func(checked bool) {
		// TODO: Implement
	})
	notifications.SetChecked(true)

	// Connection info (if connected)
	connectionInfo := widget.NewLabel("No active connection")
	if t.connected && t.shareablePort != "" {
		shareableLink := fmt.Sprintf("link.tatbeeb.sa:%s", t.shareablePort)
		connectionInfo.SetText(fmt.Sprintf("Status: Connected\nLink: %s\nDatabase: %s", shareableLink, t.dbName))
	}

	infoBox := container.NewVBox(
		widget.NewLabelWithStyle("Connection Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		connectionInfo,
	)

	// Buttons
	reconfigureBtn := widget.NewButton("Change Connection", func() {
		settings.Close()
		t.showSetupWizard()
	})

	viewLogsBtn := widget.NewButton("View Logs", func() {
		// TODO: Show logs
	})

	closeBtn := widget.NewButton("Close", func() {
		settings.Close()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("General Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		autoStart,
		notifications,
		widget.NewSeparator(),
		infoBox,
		widget.NewSeparator(),
		reconfigureBtn,
		viewLogsBtn,
		layout.NewSpacer(),
		closeBtn,
	)

	settings.SetContent(container.NewPadded(content))
	settings.Show()
}

func (t *TatbeebLinkApp) showHelp() {
	// Help window
	help := t.app.NewWindow("Help & Documentation")
	help.Resize(fyne.NewSize(500, 600))

	helpText := `Tatbeeb Link - User Guide

Getting Started:
1. Click "Connect" to start
2. Enter your SQL Server details:
   - Server (e.g., localhost)
   - Port (usually 1433)
   - Database name
   - Username and password
3. Click "Test Connection" to verify
4. Click "Connect" to start the tunnel

What You Get:
• A shareable TCP link (e.g., link.tatbeeb.sa:50123)
• Anyone with this link can connect to your database
• The connection stays active while the app runs
• No firewall configuration needed!

Troubleshooting:
• If connection fails, verify SQL Server is running
• Check username and password are correct
• Make sure SQL Server allows TCP/IP connections
• Try using "localhost" or "127.0.0.1" as server

Support:
Email: support@tatbeeb.sa
Documentation: https://docs.tatbeeb.sa

System Requirements:
• Windows 10 or 11
• SQL Server 2012+
• Internet connection for the relay`

	helpLabel := widget.NewLabel(helpText)
	helpLabel.Wrapping = fyne.TextWrapWord

	docsBtn := widget.NewButton("Open Documentation", func() {
		// TODO: Open browser to docs
	})

	closeBtn := widget.NewButton("Close", func() {
		help.Close()
	})

	content := container.NewBorder(
		nil,
		container.NewVBox(docsBtn, closeBtn),
		nil,
		nil,
		container.NewScroll(helpLabel),
	)

	help.SetContent(container.NewPadded(content))
	help.Show()
}

// testMSSQLConnection tests the connection to SQL Server
func (t *TatbeebLinkApp) testMSSQLConnection(server, port, database, username, password string) error {
	connString := fmt.Sprintf("server=%s;port=%s;database=%s;user id=%s;password=%s;encrypt=disable",
		server, port, database, username, password)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// startTunnel connects to relay server and gets assigned port
func (t *TatbeebLinkApp) startTunnel() (string, error) {
	// Connect to relay server with TLS
	tlsConfig := &tls.Config{
		ServerName: "link.tatbeeb.sa",
	}

	conn, err := tls.Dial("tcp", RelayServer, tlsConfig)
	if err != nil {
		return "", fmt.Errorf("failed to connect to relay: %w", err)
	}

	t.relayConn = conn

	// Send REGISTER command
	registerMsg := fmt.Sprintf("REGISTER\n")
	_, err = conn.Write([]byte(registerMsg))
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("failed to send register: %w", err)
	}

	// Read response with timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	conn.SetReadDeadline(time.Time{}) // Clear timeout

	// Parse response: "OK port:50123"
	response = strings.TrimSpace(response)
	parts := strings.Split(response, " ")
	if len(parts) < 2 || parts[0] != "OK" {
		conn.Close()
		return "", fmt.Errorf("unexpected response: %s", response)
	}

	// Extract port from "port:50123"
	portParts := strings.Split(parts[1], ":")
	if len(portParts) != 2 || portParts[0] != "port" {
		conn.Close()
		return "", fmt.Errorf("invalid port format: %s", parts[1])
	}

	shareablePort := portParts[1]

	// Start handling relay connection
	go t.handleRelayConnection()

	log.Printf("Tunnel started: Relay assigned port %s", shareablePort)

	return shareablePort, nil
}

// handleRelayConnection manages the main relay connection (just keeps it alive)
func (t *TatbeebLinkApp) handleRelayConnection() {
	defer func() {
		if t.relayConn != nil {
			t.relayConn.Close()
		}
	}()

	// This connection is just for registration and keeping the port alive
	// The actual data forwarding happens through separate connections

	reader := bufio.NewReader(t.relayConn)

	// Keep connection alive with heartbeats
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			if t.relayConn != nil {
				// Send heartbeat
				_, err := t.relayConn.Write([]byte("HEARTBEAT\n"))
				if err != nil {
					log.Printf("Failed to send heartbeat: %v", err)
					return
				}
			}
		}
	}()

	// Read any messages from relay (mostly just keep the connection alive)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Relay connection closed: %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line != "" {
			log.Printf("Relay message: %s", line)
		}
	}
}

// closeTunnel stops the tunnel and cleans up
func (t *TatbeebLinkApp) closeTunnel() error {
	if t.relayConn != nil {
		t.relayConn.Close()
		t.relayConn = nil
	}
	if t.tunnelListener != nil {
		err := t.tunnelListener.Close()
		t.tunnelListener = nil
		return err
	}
	return nil
}
