package auth

import (
	"fmt"
	"strings"
)

// ShowCookieExtractionGuide displays step-by-step instructions for extracting cookies
func ShowCookieExtractionGuide() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("📚 INSTAGRAM COOKIE EXTRACTION GUIDE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
	
	fmt.Println("This tool needs your Instagram session cookies to access the API.")
	fmt.Println("Follow these steps to extract them from your browser:")
	fmt.Println()
	
	// Browser selection
	fmt.Println("🌐 STEP 1: Open Instagram in your browser")
	fmt.Println("   - Go to https://www.instagram.com")
	fmt.Println("   - Log in with your account")
	fmt.Println("   - Make sure you can see your feed")
	fmt.Println()
	
	// Developer tools
	fmt.Println("🔧 STEP 2: Open Developer Tools")
	fmt.Println("   • Chrome/Edge/Brave: Press F12 or Ctrl+Shift+I (Cmd+Option+I on Mac)")
	fmt.Println("   • Firefox: Press F12 or Ctrl+Shift+I (Cmd+Option+I on Mac)")
	fmt.Println("   • Safari: Enable Developer menu in Preferences, then Cmd+Option+I")
	fmt.Println()
	
	// Network tab
	fmt.Println("📡 STEP 3: Go to the Network tab")
	fmt.Println("   - Click on the 'Network' tab in Developer Tools")
	fmt.Println("   - If it's empty, refresh the page (F5)")
	fmt.Println()
	
	// Find cookies
	fmt.Println("🍪 STEP 4: Find your cookies")
	fmt.Println("   METHOD A - From Network tab:")
	fmt.Println("   1. Look for any request to 'instagram.com'")
	fmt.Println("   2. Click on it")
	fmt.Println("   3. Go to 'Headers' section")
	fmt.Println("   4. Scroll to 'Request Headers'")
	fmt.Println("   5. Find the 'Cookie:' line")
	fmt.Println()
	fmt.Println("   METHOD B - From Application/Storage tab:")
	fmt.Println("   1. Go to 'Application' tab (Chrome) or 'Storage' tab (Firefox)")
	fmt.Println("   2. In the left sidebar, expand 'Cookies'")
	fmt.Println("   3. Click on 'https://www.instagram.com'")
	fmt.Println("   4. Look for these cookies in the list:")
	fmt.Println()
	
	// Cookie details
	fmt.Println("🔑 STEP 5: Copy these specific values:")
	fmt.Println("   ┌─────────────┬──────────────────────────────────────────────┐")
	fmt.Println("   │ Cookie Name │ What it looks like                          │")
	fmt.Println("   ├─────────────┼──────────────────────────────────────────────┤")
	fmt.Println("   │ sessionid   │ Long string with %3A and %2C                 │")
	fmt.Println("   │             │ Example: 12345678%3Aabcdef...               │")
	fmt.Println("   ├─────────────┼──────────────────────────────────────────────┤")
	fmt.Println("   │ csrftoken   │ 32-character string                          │")
	fmt.Println("   │             │ Example: YTQHujAgMhyveLvvuwCfw9CPI8ROAHoy   │")
	fmt.Println("   └─────────────┴──────────────────────────────────────────────┘")
	fmt.Println()
	
	// Tips
	fmt.Println("💡 TIPS:")
	fmt.Println("   • Copy the ENTIRE value (everything after the = sign)")
	fmt.Println("   • Don't include quotes or semicolons")
	fmt.Println("   • These cookies expire, so you may need to refresh them periodically")
	fmt.Println("   • Use a secondary account for scraping to avoid issues with your main account")
	fmt.Println()
	
	// Security warning
	fmt.Println("⚠️  SECURITY WARNING:")
	fmt.Println("   • These cookies give FULL access to your Instagram account")
	fmt.Println("   • NEVER share them with anyone")
	fmt.Println("   • Store them securely (this tool encrypts them)")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}

// ShowQuickExtractGuide shows a condensed version for experienced users
func ShowQuickExtractGuide() {
	fmt.Println("\n🍪 Quick Guide: F12 → Network tab → Refresh → Click any instagram.com request → Headers → Cookie")
	fmt.Println("   Need: sessionid=... and csrftoken=...")
	fmt.Println("   Type 'help' for detailed instructions")
}