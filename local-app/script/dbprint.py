import sqlite3

# Connect to the SQLite database file
conn = sqlite3.connect('./data/mindnoscape.db')

# Create a cursor object to interact with the database
cursor = conn.cursor()

# Fetch and display all tables in the database
cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
tables = cursor.fetchall()

print("Tables in the database:")
for table in tables:
    print(table[0])

# Fetch and display contents of each table
for table in tables:
    print(f"\nContent of table '{table[0]}':")
    cursor.execute(f"SELECT * FROM {table[0]}")
    rows = cursor.fetchall()

    for row in rows:
        print(row)

# Close the connection to the database
conn.close()
