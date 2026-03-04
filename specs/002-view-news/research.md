# Research: View News Feature

**Date**: 2024-11-20  
**Feature**: View News Feature  
**Purpose**: Resolve technical clarifications and validate architecture decisions

## Research Summary

All technical decisions are well-defined based on the existing codebase patterns and PRD requirements. The architecture leverages proven technologies already in use (RapidAPI integration pattern from google.go, Redis caching, React Native components).

## Technical Decisions

### 1. News API Provider

**Decision**: RapidAPI - Real-Time News Data API  
**Rationale**: 
- Existing codebase already uses RapidAPI (google.go pattern)
- RapidAPI key already configured in backend (RAPID_API_KEY)
- Real-Time News Data API provides football-specific filtering
- Supports query parameters for filtering (query, limit, time_published, country, lang)
- Good response structure with title, snippet, photo_url, source info

**Alternatives considered**: 
- NewsAPI.org (rejected: RapidAPI already integrated, consistent API pattern)
- Google News API (rejected: already used for different purpose, Real-Time News Data better for football filtering)
- Self-hosted news aggregation (rejected: not feasible, requires significant infrastructure)

### 2. Backend Architecture Pattern

**Decision**: Follow existing google.go pattern for external API integration  
**Rationale**:
- Consistent with existing codebase patterns
- Reuses existing Redis cache infrastructure
- Follows established error handling patterns
- RapidAPI key already in Config struct

**Alternatives considered**:
- Separate microservice (rejected: overkill for single endpoint, adds complexity)
- Direct frontend API calls (rejected: exposes API key, violates security)

### 3. Caching Strategy

**Decision**: Multi-layer caching (Backend Redis 15min, Frontend React Query 5min)  
**Rationale**:
- Backend cache (15min) minimizes RapidAPI calls and costs
- Frontend cache (5min) provides instant loading for users
- Matches existing cache patterns (google.go uses 30min for news)
- Balances freshness with API rate limits

**Alternatives considered**:
- Single layer caching (rejected: insufficient for API rate limits and user experience)
- Longer cache duration (rejected: news becomes stale, poor user experience)
- Shorter cache duration (rejected: increases API costs and rate limit risk)

### 4. Frontend Data Fetching

**Decision**: React Query (@tanstack/react-query)  
**Rationale**:
- Industry standard for React data fetching
- Built-in caching, refetching, and error handling
- Works seamlessly with React Native
- Supports pull-to-refresh patterns
- Automatic background refetching

**Alternatives considered**:
- Redux with custom middleware (rejected: overkill, adds complexity)
- SWR (rejected: React Query more popular, better React Native support)
- Fetch with useState/useEffect (rejected: manual caching and error handling too complex)

### 5. Article Viewing

**Decision**: react-native-webview for in-app article viewing  
**Rationale**:
- Already installed in mobile app (package.json shows version 13.15.0)
- Provides in-app browsing experience
- Users can return easily to app
- Supports external links gracefully

**Alternatives considered**:
- External browser only (rejected: poor UX, users leave app)
- Custom article renderer (rejected: too complex, news sources have varied HTML)

### 6. Image Caching

**Decision**: react-native-fast-image with disk cache  
**Rationale**:
- Efficient image loading and caching
- Reduces network usage
- Improves performance for thumbnail images
- Standard library for React Native image optimization

**Alternatives considered**:
- React Native Image component (rejected: no built-in disk caching)
- Manual image caching (rejected: too complex, reinventing wheel)

### 7. Error Handling Strategy

**Decision**: Graceful degradation with cached content fallback  
**Rationale**:
- Provides best user experience when API fails
- Shows cached content with warning indicator
- Allows offline viewing of previously loaded articles
- Matches existing error handling patterns

**Alternatives considered**:
- Show error only (rejected: poor UX, users see nothing)
- Retry without cache (rejected: wastes API calls, slow)

### 8. News Feed Placement

**Decision**: Integrate into existing HomeScreen  
**Rationale**:
- PRD specifies home screen integration
- HomeScreen already has tab navigation structure
- Can add news section above or below match tabs
- No need for separate navigation tab initially

**Alternatives considered**:
- Separate News tab (rejected: PRD specifies home screen, can add later based on engagement)
- Modal overlay (rejected: not discoverable, poor UX)

### 9. API Response Transformation

**Decision**: Transform RapidAPI response to internal format in backend  
**Rationale**:
- Decouples frontend from external API structure
- Allows easy API provider switching in future
- Consistent response format across app
- Can add computed fields (relative time, etc.)

**Alternatives considered**:
- Direct RapidAPI response (rejected: couples frontend to external API, harder to change)
- Frontend transformation (rejected: backend should handle API abstraction)

### 10. Configuration Management

**Decision**: Use existing config package with environment variables  
**Rationale**:
- RapidAPI key already in config (RAPID_API_KEY)
- Follows existing configuration patterns
- Supports environment-specific keys (staging vs production)
- Can add news-specific config if needed

**Alternatives considered**:
- Separate config file (rejected: inconsistent with existing patterns)
- Hardcoded values (rejected: violates security and flexibility)

## Unresolved Questions

None - all technical decisions are resolved based on existing codebase patterns and PRD requirements.

## Dependencies

### External Dependencies
- RapidAPI account with Real-Time News Data API subscription
- RapidAPI key configured in backend environment

### Internal Dependencies
- Existing Redis cache infrastructure
- Existing Go backend API structure
- Existing React Native mobile app structure
- react-native-webview (already installed)

### New Dependencies
- @tanstack/react-query (to be added to mobile app)
- react-native-fast-image (to be added to mobile app)

## Risk Assessment

### Low Risk
- API integration (proven pattern from google.go)
- Caching implementation (existing Redis infrastructure)
- Component development (standard React Native patterns)

### Medium Risk
- API rate limits (mitigated by aggressive caching)
- API availability (mitigated by cached content fallback)
- Image loading performance (mitigated by react-native-fast-image)

### Mitigation Strategies
- Monitor API usage and implement alerts
- Implement exponential backoff for failed requests
- Cache images aggressively
- Provide clear error messages with retry options

