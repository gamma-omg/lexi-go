CREATE TABLE words (
    id SERIAL PRIMARY KEY,
    lemma TEXT NOT NULL,
    lang VARCHAR(10) NOT NULL,
    class VARCHAR(50),
    rarity INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (lemma, lang, class)
);

CREATE TABLE definitions (
    id SERIAL PRIMARY KEY,
    word_id INT NOT NULL,
    def TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);
CREATE INDEX idx_definitions_word_id ON definitions(word_id);

CREATE TABLE user_picks (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    def_id INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (def_id) REFERENCES definitions(id) ON DELETE CASCADE,
    UNIQUE (user_id, def_id)
);
CREATE INDEX idx_user_picks_user_id_def_id ON user_picks(user_id, def_id);

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    tag TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_tags_tag ON tags(tag);

CREATE TABLE tags_map (
    id SERIAL PRIMARY KEY,
    pick_id INT NOT NULL,
    tag_id INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (pick_id) REFERENCES user_picks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    UNIQUE (pick_id, tag_id)
);
CREATE INDEX idx_tags_map_pick_id_tag_id ON tags_map(pick_id, tag_id);