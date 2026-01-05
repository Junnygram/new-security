import time
import os
import sys
import json
import redis
import psycopg2

def get_db_connection():
    retry_count = 0
    while retry_count < 10:
        try:
            conn = psycopg2.connect(
                host=os.getenv('DB_HOST'),
                port=os.getenv('DB_PORT'),
                user=os.getenv('DB_USER'),
                password=os.getenv('DB_PASSWORD'),
                dbname=os.getenv('DB_NAME')
            )
            return conn
        except Exception as e:
            print(f"Failed to connect to DB: {e}, retrying...", file=sys.stderr)
            time.sleep(2)
            retry_count += 1
    sys.exit("Could not connect to database")

def process_order(order_data):
    conn = get_db_connection()
    cur = conn.cursor()
    try:
        print(f"Processing order: {order_data}", file=sys.stderr)
        # Update status to processed
        cur.execute(
            "UPDATE orders SET status = %s WHERE id = %s",
            ('processed', order_data['order_id'])
        )
        conn.commit()
        print(f"Order {order_data['order_id']} marked as processed", file=sys.stderr)
    except Exception as e:
        print(f"Error processing order: {e}", file=sys.stderr)
    finally:
        cur.close()
        conn.close()

def main():
    print("Order Processor (Python) started...", file=sys.stderr)
    
    # Connect to Redis
    try:
        r = redis.Redis(
            host=os.getenv('REDIS_HOST'),
            port=os.getenv('REDIS_PORT'),
            decode_responses=True
        )
        p = r.pubsub()
        p.subscribe('orders')
        print("Subscribed to 'orders' channel", file=sys.stderr)
    except Exception as e:
        sys.exit(f"Failed to connect to Redis: {e}")

    while True:
        message = p.get_message()
        if message and message['type'] == 'message':
            try:
                data = json.loads(message['data'])
                process_order(data)
            except json.JSONDecodeError:
                print("Received invalid JSON", file=sys.stderr)
        time.sleep(0.1)

if __name__ == "__main__":
    main()
