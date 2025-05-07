CREATE TABLE ref_tokens (
    id SERIAL PRIMARY KEY,
    guid UUID UNIQUE NOT NULL, 
    token_hash VARCHAR NOT NULL,
    user_agent_hash VARCHAR NOT NULL,
    ip_hash varchar NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE blaclist_ref(
	id_ref_tokens int references ref_tokens(id),
	used_token varchar UNIQUE NOT NULL
);

