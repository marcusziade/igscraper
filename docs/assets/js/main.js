// Theme Management
const themeToggle = document.getElementById('themeToggle');
const prefersDarkScheme = window.matchMedia('(prefers-color-scheme: dark)');
let userHasManuallySetTheme = localStorage.getItem('userHasManuallySetTheme') === 'true';

// Get theme based on user preference or system
function getTheme() {
    // If user has manually set a theme, respect that
    if (userHasManuallySetTheme) {
        const savedTheme = localStorage.getItem('theme');
        if (savedTheme) {
            return savedTheme;
        }
    }
    // Otherwise, follow system preference
    return prefersDarkScheme.matches ? 'dark' : 'light';
}

// Apply theme
function applyTheme(theme, isManual = false) {
    document.documentElement.setAttribute('data-theme', theme);
    if (isManual) {
        localStorage.setItem('theme', theme);
        localStorage.setItem('userHasManuallySetTheme', 'true');
        userHasManuallySetTheme = true;
    }
}

// Initialize theme
applyTheme(getTheme());

// Listen for system theme changes
prefersDarkScheme.addEventListener('change', (e) => {
    // Only update if user hasn't manually set a theme
    if (!userHasManuallySetTheme) {
        applyTheme(e.matches ? 'dark' : 'light');
    }
});

// Theme toggle click handler
themeToggle.addEventListener('click', () => {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    applyTheme(newTheme, true); // Mark as manual change
});

// Add option to reset to system preference
window.resetThemeToSystem = () => {
    localStorage.removeItem('theme');
    localStorage.removeItem('userHasManuallySetTheme');
    userHasManuallySetTheme = false;
    applyTheme(prefersDarkScheme.matches ? 'dark' : 'light');
};

// Smooth scrolling for navigation links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            const navHeight = document.querySelector('.navbar').offsetHeight;
            const targetPosition = target.getBoundingClientRect().top + window.pageYOffset - navHeight - 20;
            window.scrollTo({
                top: targetPosition,
                behavior: 'smooth'
            });
        }
    });
});

// Add animation on scroll
const observerOptions = {
    threshold: 0.1,
    rootMargin: '0px 0px -100px 0px'
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.style.opacity = '1';
            entry.target.style.transform = 'translateY(0)';
            observer.unobserve(entry.target);
        }
    });
}, observerOptions);

// Observe feature cards and other elements
document.querySelectorAll('.feature-card, .installation-method, .usage-example, .stat-card').forEach(el => {
    el.style.opacity = '0';
    el.style.transform = 'translateY(20px)';
    el.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
    observer.observe(el);
});

// Terminal output is now rendered immediately - no typing animation

// Add parallax effect to hero terminal
const heroTerminal = document.querySelector('.hero-terminal');
if (heroTerminal) {
    window.addEventListener('scroll', () => {
        const scrolled = window.pageYOffset;
        const rate = scrolled * -0.2;
        heroTerminal.style.transform = `translateY(${rate}px)`;
    });
}

// Dynamic year in footer
const currentYear = new Date().getFullYear();
const footerYear = document.querySelector('.footer-bottom p');
if (footerYear) {
    footerYear.innerHTML = footerYear.innerHTML.replace('2025', currentYear);
}

// Add active state to navigation links
const sections = document.querySelectorAll('section[id]');
const navLinks = document.querySelectorAll('.nav-links a[href^="#"]');

function setActiveLink() {
    const scrollY = window.pageYOffset;
    
    sections.forEach(section => {
        const sectionHeight = section.offsetHeight;
        const sectionTop = section.offsetTop - 100;
        const sectionId = section.getAttribute('id');
        
        if (scrollY > sectionTop && scrollY <= sectionTop + sectionHeight) {
            navLinks.forEach(link => {
                link.classList.remove('active');
                if (link.getAttribute('href') === `#${sectionId}`) {
                    link.classList.add('active');
                }
            });
        }
    });
}

window.addEventListener('scroll', setActiveLink);

// Add copy functionality to code blocks
document.querySelectorAll('.code-block').forEach(block => {
    const copyButton = document.createElement('button');
    copyButton.className = 'copy-button';
    copyButton.innerHTML = '📋 Copy';
    copyButton.style.cssText = `
        position: absolute;
        top: 0.5rem;
        right: 0.5rem;
        padding: 0.25rem 0.75rem;
        background: var(--accent-primary);
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 0.875rem;
        cursor: pointer;
        opacity: 0;
        transition: opacity 0.3s ease;
    `;
    
    block.style.position = 'relative';
    block.appendChild(copyButton);
    
    block.addEventListener('mouseenter', () => {
        copyButton.style.opacity = '1';
    });
    
    block.addEventListener('mouseleave', () => {
        copyButton.style.opacity = '0';
    });
    
    copyButton.addEventListener('click', async () => {
        const code = block.querySelector('code').textContent;
        try {
            await navigator.clipboard.writeText(code);
            copyButton.innerHTML = '✅ Copied!';
            setTimeout(() => {
                copyButton.innerHTML = '📋 Copy';
            }, 2000);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    });
});

// Platform tabs functionality
const platformTabs = document.querySelectorAll('.platform-tab');
const installBlocks = {
    macos: document.getElementById('macos-install'),
    linux: document.getElementById('linux-install'),
    windows: document.getElementById('windows-install')
};

platformTabs.forEach(tab => {
    tab.addEventListener('click', () => {
        const platform = tab.getAttribute('data-platform');
        
        // Update active tab
        platformTabs.forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        
        // Show corresponding install block
        Object.keys(installBlocks).forEach(key => {
            if (installBlocks[key]) {
                installBlocks[key].classList.toggle('hidden', key !== platform);
            }
        });
    });
});

// Animated Terminal Functionality
class TerminalAnimation {
    constructor() {
        this.totalPhotos = 1337;
        this.downloaded = 1044;  // Start at 78%
        this.speed = 25.1;
        this.workers = 5;
        this.size = 2.61;
        this.startTime = Date.now();
        this.isPaused = false;
        this.statusMessages = [
            '● Downloading high quality originals...',
            '● Processing metadata...',
            '● Checking for duplicates...',
            '● Fetching next batch...',
            '● Rate limiting active...',
            '● Optimizing download queue...'
        ];
        this.currentStatus = 0;
        
        // Start with a slight delay to show initial state
        setTimeout(() => this.init(), 1000);
    }
    
    init() {
        // Start animation
        this.animate();
        setInterval(() => this.updateStatus(), 3000);
        
        // Pause on hover
        const terminal = document.querySelector('.terminal-window');
        if (terminal) {
            terminal.addEventListener('mouseenter', () => this.isPaused = true);
            terminal.addEventListener('mouseleave', () => this.isPaused = false);
        }
    }
    
    animate() {
        if (!this.isPaused && this.downloaded < this.totalPhotos) {
            // Update progress
            const increment = Math.floor(Math.random() * 3) + 1;
            this.downloaded = Math.min(this.downloaded + increment, this.totalPhotos);
            
            // Update speed (with smaller variation for smoother changes)
            const speedVariation = (Math.random() - 0.5) * 4;  // ±2 photos/min
            this.speed = Math.max(15, Math.min(35, this.speed + speedVariation));
            
            // Update size (average 2.5MB per photo)
            this.size = (this.downloaded * 2.5 / 1000).toFixed(2);
            
            // Calculate time left
            const photosLeft = this.totalPhotos - this.downloaded;
            const timeLeftMinutes = Math.floor(photosLeft / this.speed);
            const timeLeftSeconds = Math.floor((photosLeft % this.speed) * (60 / this.speed));
            
            // Update DOM
            this.updateDOM({
                downloaded: this.downloaded,
                speed: this.speed.toFixed(1),
                timeLeft: `${timeLeftMinutes}m ${timeLeftSeconds}s`,
                size: this.size,
                progress: Math.floor((this.downloaded / this.totalPhotos) * 100)
            });
        }
        
        // Continue animation with less frequent updates
        const delay = 800 + Math.random() * 700;
        setTimeout(() => this.animate(), delay);
    }
    
    updateStatus() {
        if (!this.isPaused) {
            this.currentStatus = (this.currentStatus + 1) % this.statusMessages.length;
            const statusEl = document.querySelector('.terminal-body .term-success');
            if (statusEl) {
                statusEl.textContent = this.statusMessages[this.currentStatus];
            }
        }
    }
    
    updateDOM(data) {
        // Get all term-value elements inside the grid
        const gridValues = document.querySelectorAll('.term-grid .term-value');
        
        // Update downloaded count (first value)
        if (gridValues[0]) {
            gridValues[0].textContent = data.downloaded;
        }
        
        // Update speed (second value)
        if (gridValues[1]) {
            gridValues[1].textContent = data.speed;
        }
        
        // Update time left (third value)
        if (gridValues[2]) {
            gridValues[2].textContent = data.timeLeft;
        }
        
        // Update workers (fourth value - skip, it's constant)
        
        // Update size (fifth value)
        if (gridValues[4]) {
            gridValues[4].textContent = data.size + ' GB';
        }
        
        // Update progress bar
        const progressFill = document.querySelector('.progress-fill');
        const progressEmpty = document.querySelector('.progress-empty');
        const progressPercent = document.querySelector('.progress-percent');
        
        if (progressFill && progressEmpty && progressPercent) {
            const totalChars = 42;  // Total width inside the progress bar (excluding borders)
            const fillChars = Math.floor(data.progress / 100 * totalChars);
            const emptyChars = totalChars - fillChars;
            
            progressFill.textContent = '█'.repeat(fillChars);
            progressEmpty.textContent = '░'.repeat(emptyChars);
            progressPercent.textContent = data.progress + '%';
        }
        
        // Add pulse effect on update
        const terminalBody = document.querySelector('.terminal-body');
        if (terminalBody) {
            terminalBody.style.opacity = '0.95';
            setTimeout(() => {
                terminalBody.style.opacity = '1';
            }, 100);
        }
    }
}

// Initialize terminal animation when page loads
document.addEventListener('DOMContentLoaded', () => {
    new TerminalAnimation();
});