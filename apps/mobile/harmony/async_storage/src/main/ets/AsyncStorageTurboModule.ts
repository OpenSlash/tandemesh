/*
 * Copyright (c) 2024 Huawei Device Co., Ltd. All rights reserved
 * Use of this source code is governed by a MIT license that can be
 * found in the LICENSE file.
 */

import relationalStore from '@ohos.data.relationalStore';
import { AnyThreadTurboModule, AnyThreadTurboModuleContext } from '@rnoh/react-native-openharmony/ts';
import ReactDatabaseSupplier from './ReactDatabaseSupplier';
import AsyncStorageErrorUtil from './AsyncStorageErrorUtil';
import Logger from './Logger';
import CommonConstants from './CommonConstants'
import AsyncLocalStorageUtil from './AsyncLocalStorageUtil';
import AsyncLocker from './AsyncLocker'
import { RNCAsyncStorage } from './RNCAsyncStorage';

const MAX_SQL_KEYS = 999;

type KeyValuePair = [string, string | null];

export class AsyncStorageTurboModule extends AnyThreadTurboModule implements RNCAsyncStorage.Spec {

  private table = new ReactDatabaseSupplier()
  private lock: AsyncLocker = new AsyncLocker()
  private destroyed: boolean = false
  private initPromise: Promise<void>

  constructor(ctx: AnyThreadTurboModuleContext) {
    super(ctx);
    Logger.debug(CommonConstants.TAG, `AsyncStorageTurboModule construct!`);
    this.initPromise =
      this.table.initialRdbStore(this.ctx.uiAbilityContext)
        .catch((e) => {
          Logger.error(CommonConstants.TAG, `initialRdbStore failed: ${AsyncStorageErrorUtil.getErrorMessage(e)}`);
          throw e;
        })
        .finally(() => {
          if (this.destroyed) {
            void this.table.closeRdbStore();
          }
        });
  }

  __onDestroy__(): void {
    this.destroyed = true;
    this.lock.markDestroyed();
    this.table.closeRdbStore().catch((e) => {
      Logger.warn(CommonConstants.TAG, `__onDestroy__ close rdbStore error: ${AsyncStorageErrorUtil.getErrorMessage(e)}`);
    });
  }

  private async ensureReady(): Promise<void> {
    if (this.destroyed) {
      throw new Error('AsyncStorage module destroyed');
    }
    await this.initPromise;
    if (this.destroyed) {
      throw new Error('AsyncStorage module destroyed');
    }
  }

  private invokeErrorCallback(
    callback: (error: unknown[]) => void,
    e: Object
  ): void {
    if (this.destroyed) {
      this.invokeDestroyedErrorCallback(callback);
      return;
    }
    Logger.warn(CommonConstants.TAG, AsyncStorageErrorUtil.getErrorMessage(e));
    callback([AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e))]);
  }

  private invokePairErrorCallback(
    callback: (error: unknown[], result: unknown[]) => void,
    e: Object
  ): void {
    if (this.destroyed) {
      this.invokeDestroyedPairErrorCallback(callback);
      return;
    }
    Logger.warn(CommonConstants.TAG, AsyncStorageErrorUtil.getErrorMessage(e));
    callback([AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e))], null);
  }

  private invokeDestroyedErrorCallback(
    callback: (error: unknown[]) => void
  ): void {
    callback([AsyncStorageErrorUtil.getDBError(null)]);
  }

  private invokeDestroyedPairErrorCallback(
    callback: (error: unknown[], result: unknown[]) => void
  ): void {
    callback([AsyncStorageErrorUtil.getDBError(null)], null);
  }

  /** 
   * Given an array of keys, this returns a map of (key, value) pairs for the keys found, and
   * (key, null) for the keys that haven't been found.
   */
  multiGet(keys: string[], callback: (error: unknown[], result: unknown[]) => void): void {
    Logger.debug(CommonConstants.TAG, `Module multiGet() Call!`);
    if (this.destroyed) {
      this.invokeDestroyedPairErrorCallback(callback);
      return;
    }
    let validationError = AsyncStorageErrorUtil.validateKeys(keys);
    if (validationError !== null) {
      callback([validationError], null);
      return;
    }
    this.runMultiGet(keys).then((result) => {
      if (this.destroyed) {
        this.invokeDestroyedPairErrorCallback(callback);
        return;
      }
      callback(result[0] as unknown[], result[1] as unknown[]);
    }).catch((e) => {
      this.invokePairErrorCallback(callback, e);
    })
    Logger.debug(CommonConstants.TAG, `Module multiGet() End!`);
  }

  async runMultiGet(keys: string[]): Promise<[Object, KeyValuePair[] | null]> {
    await this.ensureReady();
    return await this.lock.withLock(async () => {
      return await this.asyncMultiGet(keys);
    });
  }

  async asyncMultiGet(keys: string[]): Promise<[Object, KeyValuePair[] | null]> {
    Logger.debug(CommonConstants.TAG, `ModuleSub asyncMultiGet() Call!`);
    if (this.destroyed) {
      return [AsyncStorageErrorUtil.getDBError(null), null];
    }
    await this.table.ensureDatabase(this.ctx.uiAbilityContext);
    let columns: string[] = [ReactDatabaseSupplier.KEY_COLUMN, ReactDatabaseSupplier.VALUE_COLUMN];
    let keysRemaining: Set<string> = new Set();
    let data: KeyValuePair[] = [];
    for (let keyStart = 0; keyStart < keys.length; keyStart += MAX_SQL_KEYS) {
      let keyCount = Math.min(keys.length - keyStart, MAX_SQL_KEYS);
      let predicates = new relationalStore.RdbPredicates(ReactDatabaseSupplier.TABLE_CATALYST);
      Logger.debug(CommonConstants.TAG, `keys: ${keys}, keyStart: ${keyStart}, keyCount: ${keyCount}`)
      predicates.in("KEY", AsyncLocalStorageUtil.buildKeySelection(keys, keyStart, keyCount));
      keysRemaining.clear();
      let cursor = await this.table.rdbStore.query(predicates, columns);
      try {
        Logger.debug(CommonConstants.TAG, `ResultSet isAtFirstRow: ${cursor.isAtFirstRow}, row count: ${cursor.rowCount}`);
        if (cursor.rowCount !== keys.length) {
          for (let keyIndex = keyStart; keyIndex < keyStart + keyCount; keyIndex++) {
            keysRemaining.add(keys[keyIndex]);
          }
        }
        if (cursor.goToFirstRow()) {
          do {
            let row: KeyValuePair = [
              cursor.getString(cursor.getColumnIndex('KEY')),
              cursor.getString(cursor.getColumnIndex('VALUE')),
            ];
            data.push(row);
            keysRemaining.delete(cursor.getString(cursor.getColumnIndex('KEY')));
          } while (cursor.goToNextRow());
        }
      } catch (e) {
        Logger.warn(CommonConstants.TAG, AsyncStorageErrorUtil.getErrorMessage(e));
        return [AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e)), null];
      } finally {
        cursor.close();
      }
      for (let key of keysRemaining) {
        data.push([key, null]);
      }
      keysRemaining.clear();
    }
    return [null, data];
  }

  /** 
   * Inserts multiple (key, value) pairs. If one or more of the pairs cannot be inserted, this will
   * return AsyncLocalStorageFailure, but all other pairs will have been inserted.
   * The insertion will replace conflicting (key, value) pairs.
   */
  multiSet(kvPairs: unknown[], callback: (error: unknown[]) => void): void {
    Logger.debug(CommonConstants.TAG, `Module multiSet() Call!`);
    if (this.destroyed) {
      this.invokeDestroyedErrorCallback(callback);
      return;
    }
    let validationError = AsyncStorageErrorUtil.validateKvPairs(kvPairs);
    if (validationError !== null) {
      callback([validationError]);
      return;
    }
    this.runMultiSet(kvPairs as [string, string][]).then((error) => {
      if (this.destroyed) {
        this.invokeDestroyedErrorCallback(callback);
        return;
      }
      callback(error as unknown[]);
    }).catch((e) => {
      this.invokeErrorCallback(callback, e);
    })
    Logger.debug(CommonConstants.TAG, `Module multiSet() End!`);
  }

  async runMultiSet(keyValueArray: [string, string][]): Promise<Object> {
    await this.ensureReady();
    return await this.lock.withLock(async () => {
      return await this.asyncMultiSet(keyValueArray);
    });
  }

  async asyncMultiSet(keyValueArray: [string, string][]): Promise<Object> {
    Logger.debug(CommonConstants.TAG, `ModuleSub asyncMultiSet() Call!`);
    if (this.destroyed) {
      return AsyncStorageErrorUtil.getDBError(null);
    }
    if (keyValueArray.length === 0) {
      return null;
    }
    await this.table.ensureDatabase(this.ctx.uiAbilityContext);
    try {
      this.table.rdbStore.beginTransaction();
      for (let idx = 0; idx < keyValueArray.length; idx++) {
        let obj: relationalStore.ValuesBucket = {};
        obj.key = keyValueArray[idx][0];
        obj.value = keyValueArray[idx][1];
        await this.table.rdbStore.insert(ReactDatabaseSupplier.TABLE_CATALYST, obj, relationalStore.ConflictResolution.ON_CONFLICT_REPLACE)
        Logger.debug(CommonConstants.TAG, `key: ${obj.key}, value: ${obj.value}, Insert success!`)
      }
      this.table.rdbStore.commit();
      return null;
    } catch (e) {
      AsyncLocalStorageUtil.rollBackTransaction(this.table.rdbStore, 'asyncMultiSet');
      let e_message = `Insert is failed: ${AsyncStorageErrorUtil.getErrorMessage(e)}`
      Logger.warn(CommonConstants.TAG, e_message)
      return AsyncStorageErrorUtil.getError(null, e_message);
    }
  }

  /** 
   * Removes all rows of the keys given.
   */
  multiRemove(keys: string[], callback: (error: unknown[]) => void): void {
    Logger.debug(CommonConstants.TAG, `Module multiRemove() Call!`);
    if (this.destroyed) {
      this.invokeDestroyedErrorCallback(callback);
      return;
    }
    let validationError = AsyncStorageErrorUtil.validateKeys(keys);
    if (validationError !== null) {
      callback([validationError]);
      return;
    }
    this.runMultiRemove(keys).then((error)=>{
      if (this.destroyed) {
        this.invokeDestroyedErrorCallback(callback);
        return;
      }
      callback(error as unknown[]);
    }).catch((e)=>{
      this.invokeErrorCallback(callback, e);
    })
    Logger.debug(CommonConstants.TAG, `Module multiRemove() End!`);
  }

  async runMultiRemove(keys: string[]): Promise<Object> {
    await this.ensureReady();
    return await this.lock.withLock(async () => {
      return await this.asyncMultiRemove(keys);
    });
  }

  async asyncMultiRemove(keys: string[]): Promise<Object> {
    Logger.debug(CommonConstants.TAG, `ModuleSub asyncMultiRemove() Call!`);
    if (this.destroyed) {
      return AsyncStorageErrorUtil.getDBError(null);
    }
    if(keys.length === 0) {
      return null;
    }
    await this.table.ensureDatabase(this.ctx.uiAbilityContext);
    try {
      this.table.rdbStore.beginTransaction();
      for (let keyStart = 0; keyStart < keys.length; keyStart += MAX_SQL_KEYS) {
        let keyCount = Math.min(keys.length - keyStart, MAX_SQL_KEYS);
        let predicates = new relationalStore.RdbPredicates(ReactDatabaseSupplier.TABLE_CATALYST);
        predicates.in("KEY", AsyncLocalStorageUtil.buildKeySelection(keys, keyStart, keyCount));
        await this.table.rdbStore.delete(predicates);
        Logger.debug(CommonConstants.TAG, `keys: ${keys}, Delete success!`)
      }
      this.table.rdbStore.commit();
      return null;
    } catch (e) {
      AsyncLocalStorageUtil.rollBackTransaction(this.table.rdbStore, 'asyncMultiRemove');
      Logger.warn(CommonConstants.TAG, AsyncStorageErrorUtil.getErrorMessage(e));
      return AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e));
    }
  }

  /** 
   * Given an array of (key, value) pairs, this will merge the given values with the stored values
   * of the given keys, if they exist.
   */
  multiMerge(kvPairs: unknown[], callback: (error: unknown[]) => void): void {
    Logger.debug(CommonConstants.TAG, `Module multiMerge() Call!`);
    if (this.destroyed) {
      this.invokeDestroyedErrorCallback(callback);
      return;
    }
    let validationError = AsyncStorageErrorUtil.validateKvPairs(kvPairs);
    if (validationError !== null) {
      callback([validationError]);
      return;
    }
    this.runMultiMerge(kvPairs as [string, string][]).then((error)=>{
      if (this.destroyed) {
        this.invokeDestroyedErrorCallback(callback);
        return;
      }
      callback(error as unknown[]);
    }).catch((e)=>{
      this.invokeErrorCallback(callback, e);
    })
    Logger.debug(CommonConstants.TAG, `Module multiMerge() End!`);
  }

  async runMultiMerge(keyValueArray: [string, string][]): Promise<Object> {
    await this.ensureReady();
    return await this.lock.withLock(async () => {
      return await this.asyncMultiMerge(keyValueArray);
    });
  }

  async asyncMultiMerge(keyValueArray: [string, string][]): Promise<Object> {
    Logger.debug(CommonConstants.TAG, `ModuleSub asyncMultiMerge() Call!`);
    if (this.destroyed) {
      return AsyncStorageErrorUtil.getDBError(null);
    }
    await this.table.ensureDatabase(this.ctx.uiAbilityContext);
    try {
      for (let idx = 0; idx < keyValueArray.length; idx++) {
        let success = await AsyncLocalStorageUtil.mergeImpl(
          this.table.rdbStore, keyValueArray[idx][0], keyValueArray[idx][1]);
        if (!success) {
          return AsyncStorageErrorUtil.getDBError(null);
        }
      }
    } catch (e) {
      Logger.warn(CommonConstants.TAG, AsyncStorageErrorUtil.getErrorMessage(e));
      return AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e));
    }
    return null;
  }

  clear(callback: (error: unknown[]) => void): void {
    Logger.debug(CommonConstants.TAG, `Module clear() Call!`);
    if (this.destroyed) {
      let dbError: Object = AsyncStorageErrorUtil.getDBError(null);
      callback(dbError as unknown[]);
      return;
    }
    this.runClear().then((error)=>{
      if (this.destroyed) {
        let dbError: Object = AsyncStorageErrorUtil.getDBError(null);
        callback(dbError as unknown[]);
        return;
      }
      callback(error as unknown[]);
    }).catch((e)=>{
      Logger.warn(CommonConstants.TAG, AsyncStorageErrorUtil.getErrorMessage(e));
      let err: Object = AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e));
      callback(err as unknown[]);
    })
    Logger.debug(CommonConstants.TAG, `Module clear() End!`);
  }

  async runClear(): Promise<Object> {
    await this.ensureReady();
    return await this.lock.withLock(async () => {
      return await this.asyncClear();
    });
  }

  async asyncClear(): Promise<Object> {
    Logger.debug(CommonConstants.TAG, `ModuleSub asyncClear() Call!`);
    if (this.destroyed) {
      return AsyncStorageErrorUtil.getDBError(null);
    }
    await this.table.ensureDatabase(this.ctx.uiAbilityContext);
    try {
      await this.table.deleteRdbStore(this.ctx.uiAbilityContext);
    } catch (e) {
      Logger.warn(CommonConstants.TAG, `Module clear() error: ${AsyncStorageErrorUtil.getErrorMessage(e)}`);
      return AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e))
    }
    return null;
  }

  getAllKeys(callback: (error: unknown[], result: unknown[]) => void): void {
    Logger.debug(CommonConstants.TAG, `Module getAllKeys() Call!`);
    if (this.destroyed) {
      let dbError: Object = AsyncStorageErrorUtil.getDBError(null);
      callback(dbError as unknown[], null);
      return;
    }
    this.runGetAllKeys().then((result)=>{
      if (this.destroyed) {
        let dbError: Object = AsyncStorageErrorUtil.getDBError(null);
        callback(dbError as unknown[], null);
        return;
      }
      callback(result[0] as unknown[], result[1] as unknown[]);
    }).catch((e)=>{
      Logger.warn(CommonConstants.TAG, AsyncStorageErrorUtil.getErrorMessage(e));
      let err: Object = AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e));
      callback(err as unknown[], null);
    })
    Logger.debug(CommonConstants.TAG, `Module getAllKeys() End!`);
  }

  async runGetAllKeys(): Promise<[Object, string[] | null]> {
    await this.ensureReady();
    return await this.lock.withLock(async () => {
      return await this.asyncGetAllKeys();
    });
  }

  async asyncGetAllKeys(): Promise<[Object, string[] | null]> {
    Logger.debug(CommonConstants.TAG, `ModuleSub asyncGetAllKeys() Call!`);
    if (this.destroyed) {
      return [AsyncStorageErrorUtil.getDBError(null), null];
    }
    await this.table.ensureDatabase(this.ctx.uiAbilityContext);
    let data: string[] = [];
    let predicates = new relationalStore.RdbPredicates(ReactDatabaseSupplier.TABLE_CATALYST);
    predicates.isNotNull(ReactDatabaseSupplier.KEY_COLUMN);
    let cursor = await this.table.rdbStore.query(predicates);
    try {
      if(cursor.goToFirstRow()){
        do {
          data.push(cursor.getString(cursor.getColumnIndex('KEY')));
        } while (cursor.goToNextRow())
      }
    } catch (e) {
      Logger.warn(CommonConstants.TAG, `Module GetAllKeys() error: ${AsyncStorageErrorUtil.getErrorMessage(e)}`);
      return [AsyncStorageErrorUtil.getError(null, AsyncStorageErrorUtil.getErrorMessage(e)),null];
    } finally {
      cursor.close();
    }
    return [null, data]
  }
}
