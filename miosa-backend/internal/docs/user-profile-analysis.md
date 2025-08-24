# User Profile API Analysis

## Requirements
- GET /api/profile/{userId} - Retrieve user profile
- PUT /api/profile/{userId} - Update user profile
- Authentication required via JWT

## Data Model
- userId: string (UUID)
- username: string
- email: string
- bio: string (optional)
- avatar: string (URL, optional)
- createdAt: timestamp
- updatedAt: timestamp