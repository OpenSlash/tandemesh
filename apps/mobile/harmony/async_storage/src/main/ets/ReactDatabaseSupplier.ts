/*
 * Copyright (c) 2024 Huawei Device Co., Ltd. All rights reserved
 * Use of this source code is governed by a MIT license that can be
 * found in the LICENSE file.
 */

import common from '@ohos.app.ability.common'
import relationalStore from '@ohos.data.relationalStore'

import CommonConstants from './CommonConstants'
import Logger from './Logger'
import AsyncStorageErrorUtil from './AsyncStorageErrorUtil'

export default class ReactDatabaseSupplier {
  static DATABASE_NAME: string = CommonConstants.STORE_CONFIG.name;
  static TABLE_CATALYST = "catalystLocalStorage";
  static KEY_COLUMN = "key";
  static VALUE_COLUMN = "value"
  private DATABASE_VERSION = 1;
  private SLEEP_TIME_MS = 50;
  private VERSION_TABLE_CREATE =
    "CREATE TABLE IF NOT EXISTS " + ReactDatabaseSupplier.TABLE_CATALYST + " (" +
    ReactDatabaseSupplier.KEY_COLUMN + " TEXT PRIMARY KEY, " +
    ReactDatabaseSupplier.VALUE_COLUMN + " TEXT NOT NULL" +
      ")";

  rdbStore: relationalStore.RdbStore = null;

  async initialRdbStore(context: common.UIAbilityContext) {
    try {
      if(this.rdbStore !== null){
        Logger.debug(CommonConstants.TAG, 'The rdbStore exists.');
        return;
      }
      this.rdbStore = await relationalStore.getRdbStore(context, CommonConstants.STORE_CONFIG);
      this.rdbStore.version = this.DATABASE_VERSION;
      Logger.debug(CommonConstants.TAG, `Get RdbStore success`,);
      await this.rdbStore.executeSql(this.VERSION_TABLE_CREATE)
      Logger.debug(CommonConstants.TAG, `Create table done.`);
    } catch (err) {
      throw new Error(`Initial RdbStore failed: ${AsyncStorageErrorUtil.getErrorMessage(err)}`);
    }
  }

  async closeRdbStore(): Promise<void> {
    if (this.rdbStore === null) {
      return;
    }
    try {
      await this.rdbStore.close();
      Logger.debug(CommonConstants.TAG, 'Close RdbStore successfully.');
    } catch (err) {
      Logger.warn(CommonConstants.TAG, `Close RdbStore failed: ${AsyncStorageErrorUtil.getErrorMessage(err)}`);
    } finally {
      this.rdbStore = null;
    }
  }

  async deleteRdbStore(context: common.UIAbilityContext) {
    try {
      if(this.rdbStore === null){
        Logger.debug(CommonConstants.TAG, 'The rdbStore dose not exists.');
        throw new Error('The rdbStore dose not exists.');
      }
      await this.closeRdbStore();
      await relationalStore.deleteRdbStore(context, ReactDatabaseSupplier.DATABASE_NAME)
      Logger.debug(CommonConstants.TAG, `Delete RdbStore successfully.`);
    } catch (err) {
      throw new Error(`Delete RdbStore failed: ${AsyncStorageErrorUtil.getErrorMessage(err)}`);
    }
  }

  async ensureDatabase(context: common.UIAbilityContext): Promise<boolean> {
    if(this.rdbStore !== null) return true;
    for(let tries = 0; tries < 3; tries++) {
      if(this.rdbStore === null){
        try {
          await this.initialRdbStore(context);
        } catch (e) {
          throw new Error(`Ensure database failed: ${AsyncStorageErrorUtil.getErrorMessage(e)}`);
        }
        try {
          await this.sleep(this.SLEEP_TIME_MS)
        } catch (e) {
          throw new Error(`Ensure database sleep fail: ${AsyncStorageErrorUtil.getErrorMessage(e)}`)
        }
      }
    }
    if(this.rdbStore === null) {
      throw new Error(`Ensure database failed: rdbStore is null after retries`);
    }
    return true;
  }

  private sleep(delay: number): Promise<void> {
    return new Promise(resolve => {
      setTimeout(() => {
        resolve();
      }, delay);
    });
  }
}
