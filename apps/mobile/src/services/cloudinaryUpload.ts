import {ApiRequestError, makeAuthRequest} from './api';

export type CloudinaryUploadContext = 'avatar' | 'player_profile';

type SignatureResponse = {
  cloud_name: string;
  api_key: string;
  timestamp: number;
  signature?: string;
  folder: string;
  public_id?: string;
  upload_preset?: string;
  max_upload_bytes: number;
};

export type LocalImageFile = {
  uri: string;
  fileName?: string;
  mimeType?: string;
  size?: number;
};

function inferMimeType(fileName?: string): string {
  const lower = (fileName ?? '').toLowerCase();
  if (lower.endsWith('.png')) return 'image/png';
  if (lower.endsWith('.webp')) return 'image/webp';
  return 'image/jpeg';
}

async function requestSignature(
  token: string,
  context: CloudinaryUploadContext,
): Promise<SignatureResponse> {
  const data = await makeAuthRequest(token, '/upload/cloudinary/signature', 'POST', {
    body: JSON.stringify({context}),
  });
  return data as SignatureResponse;
}

export async function uploadToCloudinary(
  token: string,
  context: CloudinaryUploadContext,
  file: LocalImageFile,
): Promise<string> {
  const signature = await requestSignature(token, context);
  if (file.size != null && file.size > signature.max_upload_bytes) {
    const maxSizeMB = Math.round(signature.max_upload_bytes / (1024 * 1024));
    throw new ApiRequestError(`Image must be ${maxSizeMB} MB or smaller.`, 400);
  }
  const form = new FormData();
  form.append('api_key', signature.api_key);
  form.append('timestamp', String(signature.timestamp));
  form.append('folder', signature.folder);
  if (signature.public_id) {
    form.append('public_id', signature.public_id);
  }
  if (signature.signature) {
    form.append('signature', signature.signature);
  } else if (signature.upload_preset) {
    form.append('upload_preset', signature.upload_preset);
  } else {
    throw new ApiRequestError('Upload configuration is incomplete.', 500);
  }

  form.append('file', {
    uri: file.uri,
    name: file.fileName ?? `${context}-${Date.now()}.jpg`,
    type: file.mimeType ?? inferMimeType(file.fileName),
  } as any);

  const uploadURL = `https://api.cloudinary.com/v1_1/${signature.cloud_name}/image/upload`;
  const response = await fetch(uploadURL, {
    method: 'POST',
    body: form,
  });
  const result = (await response.json().catch(() => ({}))) as Record<string, unknown>;
  if (!response.ok) {
    const message =
      (typeof result.error === 'object' &&
      result.error &&
      typeof (result.error as {message?: unknown}).message === 'string'
        ? ((result.error as {message: string}).message)
        : null) ??
      `Upload failed (${response.status})`;
    throw new ApiRequestError(message, response.status);
  }
  const secureURL = result.secure_url;
  if (typeof secureURL !== 'string' || secureURL.trim() === '') {
    throw new ApiRequestError('Upload did not return a secure URL.', 500);
  }
  return secureURL;
}

