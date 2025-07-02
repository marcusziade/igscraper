package ui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// NotificationSender interface for platform-specific notification implementations
type NotificationSender interface {
	Send(title, message string) error
}

// LinuxNotificationSender sends notifications on Linux using notify-send
type LinuxNotificationSender struct{}

func (l *LinuxNotificationSender) Send(title, message string) error {
	cmd := exec.Command("notify-send", title, message)
	return cmd.Run()
}

// MacOSNotificationSender sends notifications on macOS using osascript
type MacOSNotificationSender struct{}

func (m *MacOSNotificationSender) Send(title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// WindowsNotificationSender sends notifications on Windows using PowerShell
type WindowsNotificationSender struct{}

func (w *WindowsNotificationSender) Send(title, message string) error {
	script := fmt.Sprintf(`
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
		$xml = @"
<toast>
	<visual>
		<binding template="ToastText02">
			<text id="1">%s</text>
			<text id="2">%s</text>
		</binding>
	</visual>
</toast>
"@
		$doc = [Windows.Data.Xml.Dom.XmlDocument]::new()
		$doc.LoadXml($xml)
		$toast = [Windows.UI.Notifications.ToastNotification]::new($doc)
		[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Instagram Scraper").Show($toast)
	`, title, message)
	
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Run()
}

// Notifier handles cross-platform notifications
type Notifier struct {
	sender NotificationSender
}

// NewNotifier creates a new Notifier based on the current platform
func NewNotifier() *Notifier {
	var sender NotificationSender
	
	switch runtime.GOOS {
	case "linux":
		sender = &LinuxNotificationSender{}
	case "darwin":
		sender = &MacOSNotificationSender{}
	case "windows":
		sender = &WindowsNotificationSender{}
	default:
		// Default to nil sender for unsupported platforms
		sender = nil
	}
	
	return &Notifier{sender: sender}
}

// SendNotification sends a desktop notification and prints to console
func (n *Notifier) SendNotification(title, message string) {
	// Always print to console
	fmt.Printf("\n%s: %s\n", Cyan(title), Yellow(message))
	
	// Send desktop notification if supported
	if n.sender != nil {
		// Ignore errors as notifications are not critical
		_ = n.sender.Send(title, message)
	}
}

// SendError sends an error notification
func (n *Notifier) SendError(title, message string) {
	fmt.Printf("\n%s: %s\n", Red(title), Red(message))
	
	if n.sender != nil {
		_ = n.sender.Send(title, message)
	}
}

// SendSuccess sends a success notification
func (n *Notifier) SendSuccess(title, message string) {
	fmt.Printf("\n%s: %s\n", Green(title), Green(message))
	
	if n.sender != nil {
		_ = n.sender.Send(title, message)
	}
}