#!/bin/bash

# Reset Database Script for Korus Music Server
# This script drops all tables and recreates them from migrations

set -e  # Exit on any error

# Load environment variables
if [ -f .env ]; then
    source .env
fi

# Default database URL if not set
if [ -z "$DATABASE_URL" ]; then
    echo "ERROR: DATABASE_URL not set. Please set it in .env file."
    echo "Example: DATABASE_URL=postgresql://user:password@host:port/database"
    exit 1
fi

echo "🗑️  Resetting Korus database..."
echo "Database URL: $DATABASE_URL"

# Extract database connection details
DB_URL="$DATABASE_URL"

echo ""
echo "⚠️  WARNING: This will DELETE ALL DATA in the database!"
echo "   - All songs, artists, albums will be removed"
echo "   - All user data, playlists, history will be removed"
echo "   - All jobs and sessions will be removed"
echo ""

# Ask for confirmation (skip if -y flag is provided)
if [ "$1" != "-y" ]; then
    read -p "Are you sure you want to continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo "Aborted."
        exit 0
    fi
fi

echo ""
echo "🔥 DROPPING ALL TABLES..."

# Connect to database and DROP EVERYTHING
psql "$DB_URL" << 'EOF'
-- Drop all tables completely
DROP TABLE IF EXISTS job_queue CASCADE;
DROP TABLE IF EXISTS user_sessions CASCADE;  
DROP TABLE IF EXISTS playlist_songs CASCADE;
DROP TABLE IF EXISTS playlists CASCADE;
DROP TABLE IF EXISTS user_library CASCADE;
DROP TABLE IF EXISTS play_history CASCADE;
DROP TABLE IF EXISTS songs CASCADE;
DROP TABLE IF EXISTS albums CASCADE;
DROP TABLE IF EXISTS artists CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS liked_songs CASCADE;
DROP TABLE IF EXISTS liked_albums CASCADE;
DROP TABLE IF EXISTS followed_artists CASCADE;
DROP TABLE IF EXISTS scan_history CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;

-- Drop all sequences
DROP SEQUENCE IF EXISTS users_id_seq CASCADE;
DROP SEQUENCE IF EXISTS artists_id_seq CASCADE;
DROP SEQUENCE IF EXISTS albums_id_seq CASCADE;
DROP SEQUENCE IF EXISTS songs_id_seq CASCADE;
DROP SEQUENCE IF EXISTS playlists_id_seq CASCADE;
DROP SEQUENCE IF EXISTS play_history_id_seq CASCADE;
DROP SEQUENCE IF EXISTS job_queue_id_seq CASCADE;

-- Show remaining tables (should be empty)
\dt

EOF

echo "✅ ALL TABLES DROPPED"

echo ""
echo "🎉 Database reset complete!"
echo ""
echo "Next steps:"
echo "1. Restart the Korus server"
echo "2. Login with your existing admin credentials"
echo "3. Trigger a library scan to re-import your music"
echo ""
echo "🚀 Ready to test batch processing!"
