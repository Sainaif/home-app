package services

// This file previously contained MongoDB-specific collation settings.
// For SQLite, case-insensitive comparisons are handled via COLLATE NOCASE
// in the schema definition, so no runtime configuration is needed here.
