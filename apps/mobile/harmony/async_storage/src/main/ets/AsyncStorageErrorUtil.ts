/*
 * Copyright (c) 2024 Huawei Device Co., Ltd. All rights reserved
 * Use of this source code is governed by a MIT license that can be
 * found in the LICENSE file.
 */

export default class AsyncStorageErrorUtil {

  static getErrorMessage(e: Object): string {
    if (e instanceof Error) {
      return e.message;
    }
    let err = e as Record<string, Object>;
    if (err.code !== undefined) {
      return `code=${String(err.code)}, message=${String(err.message ?? '')}`;
    }
    if (err.message !== undefined) {
      return String(err.message);
    }
    return String(e);
  }

  static getError(key: string | null, errorMessage: string) {
    let errorMap: Record<string, string> = {};
    errorMap["message"] = errorMessage;
    if (key !== null) {
      errorMap["key"] = key;
    }
    return errorMap;
  }

  static getInvalidKeyError(key: string | null): Record<string, string> {
    return AsyncStorageErrorUtil.getError(key, "Invalid key");
  }

  static getInvalidValueError(key: string | null): Record<string, string> {
    return AsyncStorageErrorUtil.getError(key, "Invalid Value");
  }

  static getDBError(key: string | null): Record<string, string> {
    return AsyncStorageErrorUtil.getError(key, "Database Error");
  }

  static validateKeys(keys: string[] | null | undefined): Record<string, string> | null {
    if (keys === null || keys === undefined) {
      return AsyncStorageErrorUtil.getInvalidKeyError(null);
    }
    if (!Array.isArray(keys)) {
      return AsyncStorageErrorUtil.getInvalidKeyError(null);
    }
    for (let idx = 0; idx < keys.length; idx++) {
      if (typeof keys[idx] !== 'string') {
        return AsyncStorageErrorUtil.getInvalidKeyError(null);
      }
    }
    return null;
  }

  static validateKvPairs(kvPairs: unknown[] | null | undefined): Record<string, string> | null {
    if (kvPairs === null || kvPairs === undefined) {
      return AsyncStorageErrorUtil.getInvalidValueError(null);
    }
    if (!Array.isArray(kvPairs)) {
      return AsyncStorageErrorUtil.getInvalidValueError(null);
    }
    for (let idx = 0; idx < kvPairs.length; idx++) {
      let pair = kvPairs[idx];
      if (!Array.isArray(pair) || pair.length !== 2) {
        return AsyncStorageErrorUtil.getInvalidValueError(null);
      }
      if (typeof pair[0] !== 'string') {
        return AsyncStorageErrorUtil.getInvalidKeyError(null);
      }
      if (typeof pair[1] !== 'string') {
        return AsyncStorageErrorUtil.getInvalidValueError(null);
      }
    }
    return null;
  }
}
