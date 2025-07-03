# AUR Submission Guide for igscraper

This guide will walk you through submitting igscraper to the AUR (Arch User Repository).

## Prerequisites

- AUR account (which you already have)
- SSH key added to your AUR account
- Git installed on your system

## Steps to Submit

### 1. Create AUR Repositories

You'll need to create two packages: `igscraper` (stable) and `igscraper-git` (development).

#### For the stable package (igscraper):

```bash
# Clone the empty AUR repository
git clone ssh://aur@aur.archlinux.org/igscraper.git aur-igscraper
cd aur-igscraper

# Copy the PKGBUILD and .SRCINFO files
cp ../PKGBUILD .
cp ../.SRCINFO .

# Add and commit the files
git add PKGBUILD .SRCINFO
git commit -m "Initial commit: igscraper 2.0.1"

# Push to AUR
git push origin master
```

#### For the git package (igscraper-git):

```bash
# Clone the empty AUR repository
git clone ssh://aur@aur.archlinux.org/igscraper-git.git aur-igscraper-git
cd aur-igscraper-git

# Copy the PKGBUILD-git as PKGBUILD and .SRCINFO-git as .SRCINFO
cp ../PKGBUILD-git PKGBUILD
cp ../.SRCINFO-git .SRCINFO

# Add and commit the files
git add PKGBUILD .SRCINFO
git commit -m "Initial commit: igscraper-git"

# Push to AUR
git push origin master
```

### 2. Test the Package Locally First

Before submitting, test that the package builds correctly:

```bash
# In the directory with PKGBUILD
makepkg -si
```

This will build and install the package locally. Make sure it works as expected.

### 3. Update Your Email in PKGBUILD

Don't forget to update the maintainer email in both PKGBUILD files:
- Replace `<your-email@example.com>` with your actual email

### 4. Maintaining the Package

After submission, you'll need to:

1. **Update for new releases**: When you release a new version:
   ```bash
   # Update pkgver in PKGBUILD
   # Regenerate .SRCINFO
   makepkg --printsrcinfo > .SRCINFO
   # Commit and push
   git add PKGBUILD .SRCINFO
   git commit -m "Update to version X.X.X"
   git push
   ```

2. **Monitor comments**: Check the AUR page for user comments and address any issues

3. **Keep it up to date**: Orphan the package if you can no longer maintain it

## Package URLs

Once submitted, your packages will be available at:
- https://aur.archlinux.org/packages/igscraper
- https://aur.archlinux.org/packages/igscraper-git

Users will be able to install with:
```bash
# Using an AUR helper like yay
yay -S igscraper
# or for git version
yay -S igscraper-git
```

## Tips

- Always test packages before pushing updates
- Use `namcap` to check for common packaging errors: `namcap PKGBUILD`
- Follow the [Arch packaging guidelines](https://wiki.archlinux.org/title/Arch_package_guidelines)
- Consider adding yourself as co-maintainer on the git version if someone else maintains the stable version

## Need Help?

- [AUR submission guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines)
- [PKGBUILD documentation](https://wiki.archlinux.org/title/PKGBUILD)
- #archlinux-aur on Libera.Chat IRC