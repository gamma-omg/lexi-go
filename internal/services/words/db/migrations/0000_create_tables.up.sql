DO $$
BEGIN
    CREATE TYPE lemma_class AS ENUM (
        'noun',
        'pronoun',
        'verb',
        'adjective',
        'adverb',
        'preposition',
        'conjunction',
        'interjection'
    );
EXCEPTION
    WHEN duplicate_object THEN 
        NULL;
END$$;

CREATE TABLE IF NOT EXISTS words (
    id SERIAL PRIMARY KEY,
    lemma TEXT NOT NULL,
    lang VARCHAR(10) NOT NULL,
    class lemma_class,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (lemma, lang, class)
);

CREATE TABLE IF NOT EXISTS definitions (
    id SERIAL PRIMARY KEY,
    word_id INT NOT NULL,
    def TEXT NOT NULL,
    rarity INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);
CREATE INDEX ON definitions(word_id);

CREATE TABLE IF NOT EXISTS user_picks (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    def_id INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (def_id) REFERENCES definitions(id) ON DELETE CASCADE,
    UNIQUE (user_id, def_id)
);
CREATE INDEX ON user_picks(user_id, id);

CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    tag TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX ON tags(tag);

CREATE TABLE IF NOT EXISTS tags_map (
    id SERIAL PRIMARY KEY,
    pick_id INT NOT NULL,
    tag_id INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (pick_id) REFERENCES user_picks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    UNIQUE (pick_id, tag_id)
);
CREATE INDEX ON tags_map(pick_id, tag_id);
CREATE INDEX ON tags_map(tag_id, pick_id);