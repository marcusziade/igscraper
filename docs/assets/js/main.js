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