# How to Extract Instagram Session Cookies

To use the Instagram scraper, you need to provide valid session cookies from a logged-in Instagram account.

## Steps to Extract Cookies:

1. **Open Instagram in your browser**
   - Go to https://www.instagram.com
   - Log in with your account

2. **Open Developer Tools**
   - Press F12 or right-click and select "Inspect"
   - Go to the "Network" tab

3. **Refresh the page**
   - Press F5 to reload Instagram

4. **Find the cookies**
   - Look for any request to instagram.com
   - Click on it and go to the "Headers" tab
   - Find the "Request Headers" section
   - Look for the "Cookie" header

5. **Extract the required values**
   - Find `sessionid=` and copy the value (everything until the next semicolon)
   - Find `csrftoken=` and copy the value (everything until the next semicolon)

## Using the Cookies:

### Method 1: Environment Variables
```bash
export IGSCRAPER_SESSION_ID="your_sessionid_here"
export IGSCRAPER_CSRF_TOKEN="your_csrftoken_here"
./igscraper scrape rachelc00k
```

### Method 2: Auth Command (Recommended)
```bash
./igscraper auth login
# Follow the prompts to enter your credentials
```

### Method 3: Update .env file
Edit the `.env` file and replace the test values:
```
IGSCRAPER_SESSION_ID=your_actual_sessionid_here
IGSCRAPER_CSRF_TOKEN=your_actual_csrftoken_here
```

## Security Notes:
- Keep your session cookies private - they provide full access to your Instagram account
- Session cookies expire after some time, so you may need to refresh them periodically
- Consider using a secondary Instagram account for scraping