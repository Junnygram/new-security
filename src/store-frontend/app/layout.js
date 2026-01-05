export const metadata = {
  title: 'Secure E-Commerce',
  description: 'A secure e-commerce application',
}

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
