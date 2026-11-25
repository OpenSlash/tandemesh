/*
 * Copyright (c) 2024 Huawei Device Co., Ltd. All rights reserved
 * Use of this source code is governed by a MIT license that can be
 * found in the LICENSE file.
 */

export default class AsyncLocker {
  private tail: Promise<void> = Promise.resolve();
  private destroyed: boolean = false;

  markDestroyed(): void {
    this.destroyed = true;
  }

  isDestroyed(): boolean {
    return this.destroyed;
  }

  async withLock<T>(fn: () => Promise<T>): Promise<T> {
    if (this.destroyed) {
      throw new Error('AsyncStorage module destroyed');
    }

    let release: () => void = () => {};
    const previous = this.tail;
    this.tail = new Promise<void>((resolve) => {
      release = resolve;
    });

    await previous;

    if (this.destroyed) {
      release();
      throw new Error('AsyncStorage module destroyed');
    }

    try {
      return await fn();
    } finally {
      release();
    }
  }
}
