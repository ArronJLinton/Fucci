# Data Model: View News Feature

**Date**: 2024-11-20  
**Feature**: View News Feature  
**Database**: N/A (external API data, no database storage)

## Entity Overview

The news feature uses external API data from RapidAPI Real-Time News Data. No database entities are required as news articles are fetched in real-time and cached temporarily. The data model focuses on API request/response structures and frontend data structures.

## API Data Structures

### 1. RapidAPI Request Parameters

**Purpose**: Parameters sent to RapidAPI Real-Time News Data API

```typescript
interface RapidAPIRequestParams {
  query: string;              // "Football"
  limit: number;              // 10
  time_published: string;     // "anytime"
  country: string;            // "US"
  lang: string;               // "en"
}
```

**Validation Rules**:
- query: Required, non-empty string
- limit: Required, integer between 1-100 (we use 10)
- time_published: Required, one of: "anytime", "past_hour", "past_day", "past_week", "past_month"
- country: Required, ISO country code (e.g., "US")
- lang: Required, ISO language code (e.g., "en")

### 2. RapidAPI Response Structure

**Purpose**: Raw response from RapidAPI Real-Time News Data API

```typescript
interface RapidAPIResponse {
  status: string;
  request_id: string;
  data: RapidAPIArticle[];
}

interface RapidAPIArticle {
  title: string;
  link: string;
  snippet?: string;
  photo_url?: string;
  published_datetime_utc: string;  // ISO 8601 format
  source_name: string;
  source_url: string;
  source_logo_url?: string;
  source_favicon_url?: string;
}
```

**Field Descriptions**:
- status: API response status (typically "ok")
- request_id: Unique request identifier for tracking
- data: Array of news articles
- title: Article headline (required)
- link: Article URL (required)
- snippet: Article preview text (optional)
- photo_url: Article thumbnail image URL (optional)
- published_datetime_utc: Publication timestamp in UTC (required)
- source_name: Publication name (required)
- source_url: Publication website URL (required)
- source_logo_url: Publication logo URL (optional)
- source_favicon_url: Publication favicon URL (optional)

### 3. Backend Internal Response

**Purpose**: Transformed response from backend API to frontend

```typescript
interface NewsAPIResponse {
  articles: NewsArticle[];
  cached: boolean;
  cachedAt?: string;  // ISO 8601 format
}

interface NewsArticle {
  id: string;                    // Generated unique ID
  title: string;
  snippet?: string;
  imageUrl?: string;
  sourceUrl: string;
  sourceName: string;
  publishedAt: string;            // ISO 8601 format
  relativeTime: string;          // "2 hours ago", "1 day ago"
}
```

**Transformation Rules**:
- id: Generated from article link (hash or UUID)
- title: Direct mapping from RapidAPI title
- snippet: Direct mapping from RapidAPI snippet (if available)
- imageUrl: Direct mapping from RapidAPI photo_url (if available)
- sourceUrl: Direct mapping from RapidAPI link
- sourceName: Direct mapping from RapidAPI source_name
- publishedAt: Direct mapping from RapidAPI published_datetime_utc
- relativeTime: Computed from publishedAt (e.g., "2h ago", "1d ago")

### 4. Frontend Data Structure

**Purpose**: Data structure used in React Native components

```typescript
interface NewsArticle {
  id: string;
  title: string;
  snippet?: string;
  imageUrl?: string;
  sourceUrl: string;
  sourceName: string;
  publishedAt: string;
  relativeTime: string;
}

interface NewsFeedState {
  articles: NewsArticle[];
  loading: boolean;
  error: Error | null;
  refreshing: boolean;
  cached: boolean;
  cachedAt?: string;
}
```

**State Management**:
- Managed by React Query hook (useNews)
- Articles array: List of news articles
- loading: Initial load state
- error: Error state if API call fails
- refreshing: Pull-to-refresh state
- cached: Whether data is from cache
- cachedAt: When cached data was fetched

## Cache Data Structures

### 1. Backend Redis Cache

**Cache Key Format**: `news:football:{timestamp}` or `news:football:latest`

**Cache Value**: JSON-serialized `NewsAPIResponse`

**TTL**: 15 minutes

**Cache Strategy**:
- Key includes timestamp for versioning
- "latest" key for most recent fetch
- Automatic expiration after 15 minutes
- Overwrite on new fetch

### 2. Frontend React Query Cache

**Cache Key**: `['news', 'football']`

**Cache Value**: `NewsAPIResponse`

**Stale Time**: 5 minutes

**Cache Time**: 15 minutes (matches backend TTL)

**Cache Strategy**:
- Automatic background refetch when stale
- Manual refetch on pull-to-refresh
- Cache persists across app sessions (if using persistent storage)

## Data Flow

### Request Flow

```
Frontend (React Query)
  ↓
Backend API Endpoint (/v1/api/news/football)
  ↓
Check Redis Cache (15min TTL)
  ↓
If cached: Return cached data
  ↓
If not cached: Call RapidAPI
  ↓
Transform RapidAPI response
  ↓
Cache in Redis
  ↓
Return to Frontend
  ↓
Cache in React Query (5min stale time)
```

### Error Flow

```
API Call Fails
  ↓
Check Redis Cache for Stale Data
  ↓
If available: Return cached data + cached flag
  ↓
If not available: Return error
  ↓
Frontend shows error state with retry option
```

## Validation Rules

### Backend Validation

1. **RapidAPI Response Validation**:
   - Status must be "ok" or valid status code
   - Data array must exist (can be empty)
   - Each article must have title, link, source_name, published_datetime_utc

2. **Transformation Validation**:
   - All required fields must be present
   - Dates must be valid ISO 8601 format
   - URLs must be valid format
   - Relative time must be computed correctly

3. **Cache Validation**:
   - Cache key must be properly formatted
   - Cache value must be valid JSON
   - TTL must be set correctly

### Frontend Validation

1. **Component Props Validation**:
   - Articles array must be array type
   - Loading/error states must be boolean
   - Article objects must match NewsArticle interface

2. **User Input Validation**:
   - Pull-to-refresh gesture must trigger refresh
   - Article tap must have valid sourceUrl

## State Transitions

### News Feed Loading States

```
Initial → Loading → Success
                ↓
            Error → Retry → Loading → Success
                              ↓
                          Error (max retries)
```

### Article Interaction States

```
Article Card → Tap → WebView Loading → Article Displayed
                    ↓
                Error → Error Message → Retry
```

## Future Enhancements

### Potential Database Entities (Future)

If news articles need to be stored for personalization or history:

```sql
CREATE TABLE user_news_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    article_id VARCHAR(255) NOT NULL,
    article_url TEXT NOT NULL,
    interaction_type VARCHAR(50) NOT NULL, -- 'viewed', 'saved', 'shared'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE saved_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    article_url TEXT NOT NULL,
    article_title TEXT NOT NULL,
    article_snippet TEXT,
    saved_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, article_url)
);
```

**Note**: These entities are not part of the current implementation but may be added in future phases for personalization and user history features.

