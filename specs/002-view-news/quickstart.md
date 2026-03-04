# Quickstart Guide: View News Feature

**Date**: 2024-11-20  
**Feature**: View News Feature  
**Target**: Developers setting up the development environment for news feature implementation

## Prerequisites

### System Requirements

- **Node.js**: 18.x or later
- **Go**: 1.22 or later
- **PostgreSQL**: 15 or later (for existing database, not required for news feature)
- **Redis**: 7.x or later (required for caching)
- **Docker**: 20.x or later (optional, for containerized development)
- **Git**: 2.x or later

### Development Tools

- **VS Code** (recommended) with extensions:
  - Go
  - TypeScript and JavaScript Language Features
  - React Native Tools
  - YAML (for OpenAPI specs)
- **Expo CLI**: `npm install -g @expo/cli`
- **Go tools**: `go install golang.org/x/tools/gopls@latest`

### External Services

- **RapidAPI Account**: Required for news API access
  - Sign up at https://rapidapi.com
  - Subscribe to "Real-Time News Data" API
  - Obtain API key

## Branch Setup

### Create Feature Branch

```bash
# From main branch
git checkout main
git pull origin main

# Create feature branch
git checkout -b 002-view-news

# Push branch to remote
git push -u origin 002-view-news
```

## Backend Setup

### 1. Environment Configuration

Add RapidAPI key to backend configuration (if not already present):

```bash
# In services/api/.env or environment variables
RAPID_API_KEY=your_rapidapi_key_here
REDIS_URL=redis://localhost:6379
```

**Note**: RapidAPI key may already be configured if other RapidAPI integrations exist.

### 2. Install Dependencies

Backend dependencies should already be installed. Verify:

```bash
cd services/api
go mod tidy
go mod verify
```

### 3. Start Redis

```bash
# Using Docker
docker run -d -p 6379:6379 redis:7-alpine

# Or using local Redis installation
redis-server
```

### 4. Test Backend Setup

```bash
cd services/api

# Run backend server
go run main.go

# In another terminal, test health endpoint
curl http://localhost:8080/v1/api/health

# Test Redis health
curl http://localhost:8080/v1/api/health/redis
```

## Frontend Setup

### 1. Install Dependencies

```bash
cd apps/mobile

# Install React Query
yarn add @tanstack/react-query

# Install image caching library
yarn add react-native-fast-image

# Verify existing dependencies
yarn install
```

**Note**: `react-native-webview` should already be installed (check package.json).

### 2. Configure React Query

Create or update `apps/mobile/src/config/queryClient.ts`:

```typescript
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      cacheTime: 15 * 60 * 1000, // 15 minutes
      retry: 3,
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
    },
  },
});
```

### 3. Setup React Query Provider

Update `apps/mobile/App.tsx` to wrap app with QueryClientProvider:

```typescript
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './src/config/queryClient';

// Wrap your app component
<QueryClientProvider client={queryClient}>
  {/* Your app components */}
</QueryClientProvider>
```

### 4. Test Frontend Setup

```bash
cd apps/mobile

# Start Expo development server
yarn start

# Or for specific platform
yarn ios
yarn android
```

## Development Workflow

### 1. Backend Development

#### Create News Service Package

```bash
cd services/api
mkdir -p internal/news
```

#### File Structure

```
services/api/internal/news/
├── client.go          # RapidAPI client wrapper
├── cache.go           # Cache key management
└── transformer.go     # Response transformation
```

#### Implementation Order

1. **client.go**: Implement RapidAPI HTTP client
2. **transformer.go**: Implement response transformation
3. **cache.go**: Implement cache key generation
4. **api/news.go**: Implement HTTP handler
5. **main.go**: Register news routes

#### Testing Backend

```bash
cd services/api

# Run tests
go test ./internal/news/...
go test ./internal/api/...

# Test endpoint manually
curl http://localhost:8080/v1/api/news/football
```

### 2. Frontend Development

#### Create News Components

```bash
cd apps/mobile/src
mkdir -p components/news
```

#### File Structure

```
apps/mobile/src/
├── components/news/
│   ├── NewsFeed.tsx
│   ├── NewsCard.tsx
│   ├── NewsCardSkeleton.tsx
│   └── NewsError.tsx
├── services/
│   └── newsService.ts
└── hooks/
    └── useNews.ts
```

#### Implementation Order

1. **newsService.ts**: API client for news endpoint
2. **useNews.ts**: React Query hook for news data
3. **NewsCardSkeleton.tsx**: Loading skeleton component
4. **NewsError.tsx**: Error state component
5. **NewsCard.tsx**: Individual article card
6. **NewsFeed.tsx**: Main feed container
7. **HomeScreen.tsx**: Integrate NewsFeed component

#### Testing Frontend

```bash
cd apps/mobile

# Type checking
yarn type-check

# Run tests (when tests are written)
yarn test
```

## API Testing

### Test RapidAPI Integration

```bash
# Test RapidAPI directly (replace YOUR_API_KEY)
curl -X GET "https://real-time-news-data.p.rapidapi.com/search?query=Football&limit=10&time_published=anytime&country=US&lang=en" \
  -H "X-RapidAPI-Key: YOUR_API_KEY" \
  -H "X-RapidAPI-Host: real-time-news-data.p.rapidapi.com"
```

### Test Backend Endpoint

```bash
# Test backend news endpoint
curl http://localhost:8080/v1/api/news/football

# Test with cache (second request should be faster)
curl http://localhost:8080/v1/api/news/football
```

## Common Issues & Solutions

### Issue: RapidAPI Key Not Working

**Solution**:
- Verify key is correct in environment variables
- Check RapidAPI subscription status
- Verify API endpoint URL is correct
- Check rate limits in RapidAPI dashboard

### Issue: Redis Connection Failed

**Solution**:
- Verify Redis is running: `redis-cli ping`
- Check REDIS_URL environment variable
- Verify Redis URL format: `redis://localhost:6379`
- Check firewall/network settings

### Issue: React Query Not Caching

**Solution**:
- Verify QueryClientProvider is wrapping app
- Check staleTime and cacheTime configuration
- Verify query key is consistent
- Check React Query DevTools for cache state

### Issue: Images Not Loading

**Solution**:
- Verify react-native-fast-image is installed
- Check image URLs are valid
- Verify network permissions (Android)
- Check image URL format (HTTPS required)

## Development Checklist

### Backend

- [ ] RapidAPI key configured in environment
- [ ] Redis running and accessible
- [ ] News service package created
- [ ] RapidAPI client implemented
- [ ] Response transformer implemented
- [ ] Cache implementation working
- [ ] HTTP handler implemented
- [ ] Routes registered in main.go
- [ ] Unit tests written
- [ ] Integration tests passing
- [ ] API endpoint tested manually

### Frontend

- [ ] React Query installed and configured
- [ ] react-native-fast-image installed
- [ ] News service created
- [ ] useNews hook implemented
- [ ] NewsCard component created
- [ ] NewsFeed component created
- [ ] Loading skeleton implemented
- [ ] Error state implemented
- [ ] HomeScreen integration complete
- [ ] Component tests written
- [ ] Manual testing on iOS/Android

### Integration

- [ ] Backend endpoint returns correct data
- [ ] Frontend displays articles correctly
- [ ] Caching works (check network tab)
- [ ] Pull-to-refresh works
- [ ] Article links open correctly
- [ ] Error handling works (simulate API failure)
- [ ] Offline mode works (cached content)
- [ ] Performance meets targets (<3s load)

## Next Steps

After completing the quickstart:

1. Review [plan.md](./plan.md) for implementation details
2. Review [data-model.md](./data-model.md) for data structures
3. Review [contracts/api.yaml](./contracts/api.yaml) for API specification
4. Follow implementation tasks (to be generated by `/speckit.tasks`)

## Resources

- [RapidAPI Real-Time News Data Documentation](https://rapidapi.com/letscrape-6bRBa3QguO5/api/real-time-news-data)
- [React Query Documentation](https://tanstack.com/query/latest)
- [React Native WebView Documentation](https://github.com/react-native-webview/react-native-webview)
- [react-native-fast-image Documentation](https://github.com/DylanVann/react-native-fast-image)

## Support

For issues or questions:
- Check existing codebase patterns (google.go for API integration)
- Review constitution for code quality standards
- Consult team lead for architecture decisions

