PWA Icon Generation Instructions
=================================

The icon.svg file contains the Holy Home logo design (purple house with pink accent).

To generate proper PNG icons for PWA:

1. Use an online tool like:
   - https://realfavicongenerator.net/
   - https://www.favicon-generator.org/
   - ImageMagick: convert icon.svg -resize 192x192 icon-192.png

2. Generate these sizes:
   - icon-192.png (192x192)
   - icon-512.png (512x512)

3. Place them in the /public directory

Note: The current setup will work with the SVG fallback, but PNG files
are recommended for better compatibility across all devices.
