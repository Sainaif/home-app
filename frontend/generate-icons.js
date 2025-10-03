import sharp from 'sharp';
import fs from 'fs';

const sizes = [192, 512];

async function generateIcons() {
  const svgBuffer = fs.readFileSync('public/icon.svg');

  for (const size of sizes) {
    await sharp(svgBuffer)
      .resize(size, size)
      .png()
      .toFile(`public/pwa-${size}x${size}.png`);

    console.log(`Generated pwa-${size}x${size}.png`);
  }

  // Generate apple-touch-icon
  await sharp(svgBuffer)
    .resize(180, 180)
    .png()
    .toFile('public/apple-touch-icon.png');

  console.log('Generated apple-touch-icon.png');

  console.log('✅ All icons generated successfully!');
}

generateIcons().catch(console.error);
