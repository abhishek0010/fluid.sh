## Redo Existing Tag
git tag -d v0.0.2-beta
git tag -a v0.0.2-beta -m "v0.0.2-beta"
git push origin v0.0.2-beta --force

or 

## New Tag
git tag -a v0.0.2-beta -m "v0.0.2-beta"
git push origin v0.0.2-beta
