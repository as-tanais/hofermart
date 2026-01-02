CREATE TABLE IF NOT EXISTS goods_rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match TEXT NOT NULL UNIQUE,
    reward NUMERIC(10,2) NOT NULL,
    reward_type TEXT NOT NULL CHECK (reward_type IN ('%', 'pt'))
);