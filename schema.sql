-- Keke Database Schema - Phase 1
-- Run this in Supabase SQL Editor

-- PLANS TABLE
CREATE TABLE IF NOT EXISTS plans (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  monthly_credits INTEGER NOT NULL,
  price_cents INTEGER NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

INSERT INTO plans (id, name, monthly_credits, price_cents) VALUES
  ('free', 'Free', 10, 0),
  ('pro', 'Pro', 285, 2000),
  ('team', 'Team', 570, 4000)
ON CONFLICT (id) DO NOTHING;

-- USERS TABLE
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  pc_hash TEXT NOT NULL,
  plan_id TEXT NOT NULL DEFAULT 'free' REFERENCES plans(id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_pc_hash ON users(pc_hash);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- CREDITS TABLE
CREATE TABLE IF NOT EXISTS credits (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  remaining INTEGER NOT NULL DEFAULT 10,
  monthly_limit INTEGER NOT NULL DEFAULT 10,
  reset_date DATE NOT NULL DEFAULT (CURRENT_DATE + INTERVAL '1 month'),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ACTIONS TABLE (audit log)
CREATE TABLE IF NOT EXISTS actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action_type TEXT NOT NULL,
  model_used TEXT,
  credits_used INTEGER NOT NULL,
  metadata JSONB,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_actions_user_id ON actions(user_id);
CREATE INDEX IF NOT EXISTS idx_actions_created_at ON actions(created_at DESC);

-- FUNCTION: Reset monthly credits
CREATE OR REPLACE FUNCTION reset_monthly_credits()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE credits
  SET 
    remaining = monthly_limit,
    reset_date = CURRENT_DATE + INTERVAL '1 month',
    updated_at = NOW()
  WHERE reset_date <= CURRENT_DATE;
END;
$$;

-- FUNCTION: Deduct credits
CREATE OR REPLACE FUNCTION deduct_credits(
  p_user_id UUID,
  p_action_type TEXT,
  p_model_used TEXT,
  p_credits_used INTEGER,
  p_metadata JSONB DEFAULT NULL
)
RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
  current_balance INTEGER;
BEGIN
  SELECT remaining INTO current_balance
  FROM credits
  WHERE user_id = p_user_id;

  IF current_balance < p_credits_used THEN
    RETURN FALSE;
  END IF;

  UPDATE credits
  SET 
    remaining = remaining - p_credits_used,
    updated_at = NOW()
  WHERE user_id = p_user_id;

  INSERT INTO actions (user_id, action_type, model_used, credits_used, metadata)
  VALUES (p_user_id, p_action_type, p_model_used, p_credits_used, p_metadata);

  RETURN TRUE;
END;
$$;

-- ROW LEVEL SECURITY
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE credits ENABLE ROW LEVEL SECURITY;
ALTER TABLE actions ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_select_own ON users
  FOR SELECT USING (auth.uid() = id);

CREATE POLICY credits_select_own ON credits
  FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY actions_select_own ON actions
  FOR SELECT USING (auth.uid() = user_id);