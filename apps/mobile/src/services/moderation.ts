import {Alert} from 'react-native';
import {makeAuthRequest} from './api';
import type {ReportStoryReason} from './matchStoryApi';

export type ReportableType =
  | 'story'
  | 'debate_response'
  | 'avatar'
  | 'player_profile'
  | 'user';

export type ContentReportReason = ReportStoryReason;

const REPORT_REASON_OPTIONS: {reason: ContentReportReason; label: string}[] = [
  {reason: 'inappropriate_content', label: 'Inappropriate content'},
  {reason: 'harassment', label: 'Harassment or abuse'},
  {reason: 'spam', label: 'Spam'},
  {reason: 'fake_team', label: 'Fake or misleading'},
  {reason: 'other', label: 'Other'},
];

export async function createContentReport(
  token: string,
  payload: {
    reportable_type: ReportableType;
    reportable_id: string;
    reason: ContentReportReason;
    description?: string;
  },
): Promise<void> {
  await makeAuthRequest(token, '/reports', 'POST', {
    body: JSON.stringify(payload),
  });
}

export async function blockUser(
  token: string,
  payload: {
    blocked_user_id: number;
    reportable_type?: ReportableType;
    reportable_id?: string;
    reason?: ContentReportReason;
    description?: string;
  },
): Promise<void> {
  await makeAuthRequest(token, '/users/blocks', 'POST', {
    body: JSON.stringify(payload),
  });
}

/** Present reason picker, then create a content report. */
export function promptReportContent(opts: {
  token: string;
  reportableType: ReportableType;
  reportableId: string;
  title?: string;
  onSuccess?: () => void;
}): void {
  const buttons = REPORT_REASON_OPTIONS.map(opt => ({
    text: opt.label,
    onPress: () => {
      void (async () => {
        try {
          await createContentReport(opts.token, {
            reportable_type: opts.reportableType,
            reportable_id: opts.reportableId,
            reason: opt.reason,
          });
          opts.onSuccess?.();
          Alert.alert(
            'Report submitted',
            'Thanks for helping keep Fucci safe. Our team reviews reports within 24 hours.',
          );
        } catch {
          Alert.alert('Could not report', 'Please try again.');
        }
      })();
    },
  }));
  Alert.alert(
    opts.title ?? 'Report content',
    'Why are you reporting this?',
    [...buttons, {text: 'Cancel', style: 'cancel'}],
  );
}

/** Block user (auto-creates a report + emails moderation) and hide locally via onSuccess. */
export function promptBlockUser(opts: {
  token: string;
  blockedUserId: number;
  reportableType?: ReportableType;
  reportableId?: string;
  onSuccess?: () => void;
}): void {
  Alert.alert(
    'Block user?',
    'Their content will be removed from your feed immediately. We will also notify our moderation team.',
    [
      {text: 'Cancel', style: 'cancel'},
      {
        text: 'Block',
        style: 'destructive',
        onPress: () => {
          void (async () => {
            try {
              await blockUser(opts.token, {
                blocked_user_id: opts.blockedUserId,
                reportable_type: opts.reportableType,
                reportable_id: opts.reportableId,
                reason: 'harassment',
              });
              opts.onSuccess?.();
              Alert.alert(
                'User blocked',
                'You will no longer see their content. Our team has been notified.',
              );
            } catch {
              Alert.alert('Could not block user', 'Please try again.');
            }
          })();
        },
      },
    ],
  );
}

/** Combined Report / Block action sheet for UGC surfaces. */
export function promptModerationActions(opts: {
  token: string;
  targetUserId: number;
  reportableType: ReportableType;
  reportableId: string;
  /** Fired after a successful report when `onReported` is not set. */
  onReportedOrBlocked?: () => void;
  onReported?: () => void;
  /** Fired after a successful block when set; otherwise `onReportedOrBlocked`. */
  onBlocked?: () => void;
}): void {
  const afterReport = opts.onReported ?? opts.onReportedOrBlocked;
  const afterBlock = opts.onBlocked ?? opts.onReportedOrBlocked;
  Alert.alert('Safety options', 'Help keep Fucci free of abuse.', [
    {
      text: 'Report content',
      onPress: () =>
        promptReportContent({
          token: opts.token,
          reportableType: opts.reportableType,
          reportableId: opts.reportableId,
          onSuccess: afterReport,
        }),
    },
    {
      text: 'Report avatar',
      onPress: () =>
        promptReportContent({
          token: opts.token,
          reportableType: 'avatar',
          reportableId: String(opts.targetUserId),
          title: 'Report avatar',
          onSuccess: afterReport,
        }),
    },
    {
      text: 'Block user',
      style: 'destructive',
      onPress: () =>
        promptBlockUser({
          token: opts.token,
          blockedUserId: opts.targetUserId,
          reportableType: opts.reportableType,
          reportableId: opts.reportableId,
          onSuccess: afterBlock,
        }),
    },
    {text: 'Cancel', style: 'cancel'},
  ]);
}

export function isAccountDeactivatedError(payload: {
  status?: number;
  code?: string;
  message?: string;
}): boolean {
  if (payload.code === 'ACCOUNT_INACTIVE') {
    return true;
  }
  if (payload.status === 403 && /deactivated/i.test(payload.message ?? '')) {
    return true;
  }
  return false;
}
