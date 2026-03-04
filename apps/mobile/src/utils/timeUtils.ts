/**
 * Time utility functions for formatting dates and relative times
 */

/**
 * Formats a timestamp or ISO 8601 date string into a relative time string
 *
 * Examples:
 * - "Just now" (less than 1 minute)
 * - "5m ago" (less than 1 hour)
 * - "2h ago" (less than 1 day)
 * - "1d ago" (less than 1 week)
 * - "3d ago" (less than 1 month)
 * - "2 weeks ago" (less than 1 month)
 * - "1 month ago" (less than 1 year)
 * - "2 years ago" (1 year or more)
 *
 * @param dateString - ISO 8601 date string or timestamp
 * @returns Human-readable relative time string
 */
export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  if (Number.isNaN(date.getTime())) {
    return 'Unknown';
  }
  const now = new Date();
  const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  // Less than 1 minute
  if (diffInSeconds < 60) {
    return 'Just now';
  }

  // Less than 1 hour
  if (diffInSeconds < 3600) {
    const minutes = Math.floor(diffInSeconds / 60);
    return `${minutes}m ago`;
  }

  // Less than 1 day
  if (diffInSeconds < 86400) {
    const hours = Math.floor(diffInSeconds / 3600);
    return `${hours}h ago`;
  }

  // Less than 1 week
  if (diffInSeconds < 604800) {
    const days = Math.floor(diffInSeconds / 86400);
    return `${days}d ago`;
  }

  // Less than 1 month (approximately 30 days)
  if (diffInSeconds < 2592000) {
    const weeks = Math.floor(diffInSeconds / 604800);
    if (weeks === 1) {
      return '1 week ago';
    }
    return `${weeks} weeks ago`;
  }

  // Less than 1 year
  if (diffInSeconds < 31536000) {
    const months = Math.floor(diffInSeconds / 2592000);
    if (months === 1) {
      return '1 month ago';
    }
    return `${months} months ago`;
  }

  // 1 year or more
  const years = Math.floor(diffInSeconds / 31536000);
  if (years === 1) {
    return '1 year ago';
  }
  return `${years} years ago`;
}
