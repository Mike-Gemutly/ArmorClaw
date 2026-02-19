-- PostgreSQL Initialization Script for ArmorClaw Matrix
-- Creates databases for Matrix homeserver and License server

-- Ensure proper encoding
SET client_encoding = 'UTF8';

-- Create matrix database (if not exists via environment)
-- This will be created by POSTGRES_DB, but we can add extensions here

\connect matrix;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- For text search
CREATE EXTENSION IF NOT EXISTS btree_gin; -- For composite indexes

-- Create license database for license-server
CREATE DATABASE licenses;

\connect licenses;

-- Enable extensions for license server
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE matrix TO armorclaw;
GRANT ALL PRIVILEGES ON DATABASE licenses TO armorclaw;

-- Log initialization
DO $$
BEGIN
    RAISE NOTICE 'ArmorClaw databases initialized at %', NOW();
END
$$;
