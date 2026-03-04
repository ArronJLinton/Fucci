# Tasks: View News Feature

**Feature**: View News Feature  
**Branch**: `002-view-news`  
**Date**: 2024-11-20  
**Generated from**: spec.md, data-model.md, contracts/api.yaml, plan.md

## Summary

This task list implements a news feed feature that displays recent and historic football news on the Fucci home screen. The feature integrates RapidAPI's Real-Time News Data API through a Go backend service with Redis caching, and displays articles in a React Native mobile app using React Query for data fetching and caching.

**Total Tasks**: 37  
**User Stories**: 1 (P0)  
**MVP Scope**: User Story 1 (View News Feed) - 25 tasks (Phases 1-3)  
**Full Implementation**: All 37 tasks (includes polish phase)

## Dependencies

**Story Completion Order**:

1. **Phase 1**: Setup (dependencies and configuration)
2. **Phase 2**: Foundational (shared utilities if needed)
3. **Phase 3**: User Story 1 - View News Feed (P0) - **MVP**
4. **Phase 4**: Polish & Cross-Cutting Concerns

**Parallel Opportunities**: Backend and frontend can be developed in parallel after Phase 1. Within backend, client, cache, and transformer can be developed in parallel. Within frontend, components can be developed in parallel.

## Implementation Strategy

**MVP First**: Focus on User Story 1 (View News Feed) to deliver core value proposition. This is the only user story and represents the complete feature.

**Incremental Delivery**: Backend can be completed and tested independently before frontend integration.

**Independent Test Criteria**:

- **US1**: Can open the home screen, view the news feed with up to 10 articles, tap articles to open in web view, and refresh the feed

---

## Phase 1: Setup (Project Initialization)

**Goal**: Install dependencies and configure development environment

**Independent Test**: Dependencies installed and configuration verified

### Setup Tasks

- [x] T001 Install @tanstack/react-query in apps/mobile/package.json
- [x] T002 Install react-native-fast-image in apps/mobile/package.json
- [x] T003 Verify react-native-webview is installed in apps/mobile/package.json
- [x] T004 Create React Query configuration in apps/mobile/src/config/queryClient.ts
- [x] T005 Integrate QueryClientProvider in apps/mobile/App.tsx
- [x] T006 Verify RapidAPI key is configured in services/api environment variables

---

## Phase 2: Foundational (Blocking Prerequisites)

**Goal**: Create shared utilities and types needed for news feature

**Independent Test**: Types and utilities can be imported and used

### Foundational Tasks

- [x] T007 Create TypeScript types for news data in apps/mobile/src/types/news.ts
- [x] T008 Create utility function for relative time formatting in apps/mobile/src/utils/timeUtils.ts
- [x] T009 Create utility function for generating article ID from URL in services/api/internal/news/utils.go

---

## Phase 3: User Story 1 - View News Feed (P0) - MVP

**Goal**: Users can view recent and historic football news on the home screen to stay informed about the latest happenings in world football.

**Independent Test**: Can be fully tested by opening the home screen, viewing the news feed, and interacting with articles without any other features.

### Backend Implementation

#### News Service Package

- [x] T010 [P] [US1] Create news service package directory structure in services/api/internal/news/
- [x] T011 [P] [US1] Implement RapidAPI client wrapper in services/api/internal/news/client.go
- [x] T012 [P] [US1] Implement cache key management in services/api/internal/news/cache.go
- [x] T013 [P] [US1] Implement response transformer in services/api/internal/news/transformer.go
- [x] T014 [US1] Create news HTTP handler in services/api/internal/api/news.go
- [x] T015 [US1] Register news routes in services/api/main.go

### Frontend Implementation

#### News Service & Hook

- [ ] T016 [P] [US1] Create news API service client in apps/mobile/src/services/newsService.ts
- [ ] T017 [US1] Create useNews custom hook in apps/mobile/src/hooks/useNews.ts

#### News Components

- [ ] T018 [P] [US1] Create NewsCardSkeleton loading component in apps/mobile/src/components/news/NewsCardSkeleton.tsx
- [ ] T019 [P] [US1] Create NewsError error state component in apps/mobile/src/components/news/NewsError.tsx
- [ ] T020 [P] [US1] Create NewsEmpty empty state component in apps/mobile/src/components/news/NewsEmpty.tsx
- [ ] T021 [P] [US1] Create NewsCard article card component in apps/mobile/src/components/news/NewsCard.tsx
- [ ] T022 [US1] Create NewsFeed main container component in apps/mobile/src/components/news/NewsFeed.tsx

#### HomeScreen Integration

- [ ] T023 [US1] Integrate NewsFeed component into HomeScreen in apps/mobile/src/screens/HomeScreen.tsx
- [ ] T024 [US1] Create NewsWebViewScreen for article viewing in apps/mobile/src/screens/NewsWebViewScreen.tsx (if not exists)
- [ ] T025 [US1] Add navigation to NewsWebViewScreen from NewsCard in apps/mobile/src/components/news/NewsCard.tsx

---

## Phase 4: Polish & Cross-Cutting Concerns

**Goal**: Enhance user experience, error handling, and performance

**Independent Test**: All edge cases handled gracefully, performance targets met

### Polish Tasks

- [ ] T026 Add analytics tracking for article opens in apps/mobile/src/components/news/NewsCard.tsx
- [ ] T027 Implement image error fallback with placeholder icon in apps/mobile/src/components/news/NewsCard.tsx
- [ ] T028 Add pull-to-refresh functionality to NewsFeed in apps/mobile/src/components/news/NewsFeed.tsx
- [ ] T029 Implement exponential backoff for failed API requests in services/api/internal/news/client.go
- [ ] T030 Add comprehensive error logging in services/api/internal/api/news.go
- [ ] T031 Verify cache TTL configuration (15min backend, 5min frontend) matches requirements
- [ ] T032 Add accessibility labels to all news components (WCAG 2.1 AA compliance)
- [ ] T033 Performance testing: Verify <3s load time on 4G connection
- [ ] T034 Add unit tests for news service package in services/api/internal/news/
- [ ] T035 Add unit tests for news components in apps/mobile/src/components/news/**tests**/
- [ ] T036 Add integration tests for news API endpoint in services/api/internal/api/
- [ ] T037 Add E2E test for news feed user journey in apps/mobile/**tests**/

---

## Task Summary

**By Phase**:

- Phase 1 (Setup): 6 tasks
- Phase 2 (Foundational): 3 tasks
- Phase 3 (User Story 1): 16 tasks
- Phase 4 (Polish): 12 tasks

**MVP Tasks (Phases 1-3)**: 25 tasks

**By Component**:

- Backend: 6 tasks
- Frontend: 19 tasks

**Parallel Opportunities**:

- Backend service package files (T011-T013) can be developed in parallel
- Frontend components (T018-T021) can be developed in parallel
- Backend and frontend can be developed in parallel after Phase 1

**Critical Path**:

1. Phase 1 (Setup) → Must complete first
2. Phase 2 (Foundational) → Can start after Phase 1
3. Backend implementation (T010-T015) → Can start after Phase 2
4. Frontend service & hook (T016-T017) → Can start after Phase 2
5. Frontend components (T018-T022) → Depends on T016-T017
6. HomeScreen integration (T023-T025) → Depends on T022
7. Phase 4 (Polish) → Depends on Phase 3 completion
