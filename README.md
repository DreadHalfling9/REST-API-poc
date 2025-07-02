Requirements:
You must have an active PostgreSQL database running to use this Todo app.
All data is stored and retrieved from a todos table.

Setup:
To use this app, run the following SQL command in your PostgreSQL environment to create the required table:

CREATE TABLE todos (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT FALSE
);
