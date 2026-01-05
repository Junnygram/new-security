'use client';

import React, { useState } from 'react';

export default function Home() {
  const [status, setStatus] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const data = {
      product_id: formData.get('product_id'),
      quantity: parseInt(formData.get('quantity'), 10),
    };

    try {
      const response = await fetch('/api/buy', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });

      if (response.ok) {
        const result = await response.json();
        setStatus(`Success: ${result.status} (Order ID: ${result.order_id})`);
      } else {
        setStatus('Error: Failed to process order');
      }
    } catch (err) {
      setStatus('Error: Network error');
    }
  };

  return (
    <main style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <h1>Secure E-Commerce Store</h1>
      <p>Enter details to purchase an item.</p>
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: '1rem' }}>
          <label>Product ID: </label>
          <input type="text" name="product_id" placeholder="12345" defaultValue="test-product" />
        </div>
        <div style={{ marginBottom: '1rem' }}>
          <label>Quantity: </label>
          <input type="number" name="quantity" defaultValue="1" />
        </div>
        <button type="submit" style={{ padding: '0.5rem 1rem', background: 'blue', color: 'white' }}>Buy Now</button>
      </form>
      {status && <p style={{ marginTop: '1rem', fontWeight: 'bold' }}>{status}</p>}
    </main>
  );
}
