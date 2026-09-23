# Issue 002

## Problem
Users passwords are stored in plain text in the database. This is not acceptable for production.
We still need someway of inserting users manually since we don't want anyone to be able to register and use the application.

## Solution
Only store hashes in the db.

## Definition of Done
- Passwords are only stored as hashes in the database
- We still have a way of inserting users.
- All existing test still pass and all new functionality have tests.
