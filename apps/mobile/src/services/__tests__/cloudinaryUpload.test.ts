/// <reference types="jest" />

import * as api from '../api';
import {uploadToCloudinary} from '../cloudinaryUpload';

describe('cloudinaryUpload', () => {
  let previousFetch: typeof globalThis.fetch;

  beforeEach(() => {
    previousFetch = globalThis.fetch;
  });

  afterEach(() => {
    jest.restoreAllMocks();
    globalThis.fetch = previousFetch;
  });

  it('rejects file larger than max_upload_bytes', async () => {
    jest.spyOn(api, 'makeAuthRequest').mockResolvedValue({
      cloud_name: 'demo',
      api_key: 'key',
      timestamp: 123456,
      signature: 'sig',
      folder: 'fucci/avatars',
      max_upload_bytes: 5 * 1024 * 1024,
    } as never);

    await expect(
      uploadToCloudinary('token', 'avatar', {
        uri: 'file://avatar.jpg',
        size: 6 * 1024 * 1024,
      }),
    ).rejects.toThrow('5 MB or smaller');
  });

  it('uploads and returns secure_url', async () => {
    jest.spyOn(api, 'makeAuthRequest').mockResolvedValue({
      cloud_name: 'demo',
      api_key: 'key',
      timestamp: 123456,
      signature: 'sig',
      folder: 'fucci/avatars',
      max_upload_bytes: 5 * 1024 * 1024,
    } as never);

    globalThis.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({secure_url: 'https://res.cloudinary.com/demo/image/upload/v1/fucci/avatars/a.jpg'}),
    } as never);

    const url = await uploadToCloudinary('token', 'avatar', {
      uri: 'file://avatar.jpg',
      fileName: 'avatar.jpg',
      mimeType: 'image/jpeg',
      size: 1024,
    });

    expect(url).toContain('res.cloudinary.com');
    expect(globalThis.fetch).toHaveBeenCalledTimes(1);
  });
});

