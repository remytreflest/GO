CREATE TABLE note_tags (
    note_id TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (note_id, tag)
);

CREATE INDEX note_tags_tag_idx ON note_tags (tag);
