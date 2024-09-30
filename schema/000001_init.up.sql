CREATE TABLE songs (
    id SERIAL PRIMARY KEY,
    group_name VARCHAR(255),
    song_name VARCHAR(255),
    release_date DATE,
    song_text TEXT,
    link TEXT
);