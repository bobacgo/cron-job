<template>
  <t-form
    ref="form"
    class="item-container"
    :class="[`login-${type}`]"
    :data="formData"
    :rules="FORM_RULES"
    label-width="0"
    @submit="onSubmit"
  >
    <template v-if="type === 'password'">
      <t-form-item name="account">
        <t-input 
          v-model="formData.account" 
          size="large" 
          :placeholder="`${t('pages.login.input.account')}：admin`"
          clearable
        >
          <template #prefix-icon>
            <t-icon name="user" />
          </template>
        </t-input>
      </t-form-item>

      <t-form-item name="password">
        <t-input
          v-model="formData.password"
          size="large"
          :type="showPsw ? 'text' : 'password'"
          clearable
          :placeholder="`${t('pages.login.input.password')}：admin`"
        >
          <template #prefix-icon>
            <t-icon name="lock-on" />
          </template>
          <template #suffix-icon>
            <t-icon :name="showPsw ? 'browse' : 'browse-off'" class="icon-btn" @click="showPsw = !showPsw" />
          </template>
        </t-input>
      </t-form-item>

      <div class="check-container remember-pwd">
        <t-checkbox>{{ t('pages.login.remember') }}</t-checkbox>
        <span class="tip">{{ t('pages.login.forget') }}</span>
      </div>
    </template>

    <!-- 扫码登录 -->
    <template v-else-if="type === 'qrcode'">
      <div class="tip-container">
        <span class="tip">{{ t('pages.login.wechatLogin') }}</span>
        <span class="refresh">{{ t('pages.login.refresh') }} <t-icon name="refresh" /> </span>
      </div>
      <qrcode-vue value="" :size="160" level="H" class="qrcode-container" />
    </template>

    <!-- 手机号登录 -->
    <template v-else>
      <t-form-item name="phone">
        <t-input v-model="formData.phone" size="large" :placeholder="t('pages.login.input.phone')" clearable>
          <template #prefix-icon>
            <t-icon name="mobile" />
          </template>
        </t-input>
      </t-form-item>

      <t-form-item class="verification-code" name="verifyCode">
        <t-input v-model="formData.verifyCode" size="large" :placeholder="t('pages.login.input.verification')" clearable />
        <t-button size="large" variant="outline" :disabled="countDown > 0" @click="sendCode">
          {{ countDown === 0 ? t('pages.login.sendVerification') : `${countDown}秒后可重发` }}
        </t-button>
      </t-form-item>
    </template>

    <t-form-item v-if="type !== 'qrcode'" class="btn-container" style="display: none">
      <t-button block size="large" type="submit"> {{ t('pages.login.signIn') }} </t-button>
    </t-form-item>
    
    <!-- 隐藏的提交按钮，用于支持回车提交 -->
    <button type="submit" style="display: none"></button>

    <div class="switch-container">
      <span v-if="type !== 'password'" class="tip" @click="switchLoginType('password')">{{
        t('pages.login.accountLogin')
      }}</span>
      <span v-if="type !== 'qrcode'" class="tip" @click="switchLoginType('qrcode')">{{ t('pages.login.wechatLogin') }}</span>
      <span v-if="type !== 'phone'" class="tip" @click="switchLoginType('phone')">{{ t('pages.login.phoneLogin') }}</span>
      <span class="tip" @click="switchToRegister">{{ t('pages.login.register') || '注册账号' }}</span>
    </div>
  </t-form>
</template>
<script setup lang="ts">
import QrcodeVue from 'qrcode.vue';
import type { FormInstanceFunctions, FormRule, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { useCounter } from '@/hooks';
import { t } from '@/locales';
import { useUserStore } from '@/store';

const userStore = useUserStore();

const emit = defineEmits<{
  switchType: [type: string];
}>();

const INITIAL_DATA = {
  phone: '',
  account: 'admin',
  password: 'admin',
  verifyCode: '',
  checked: false,
};

const FORM_RULES: Record<string, FormRule[]> = {
  phone: [{ required: true, message: t('pages.login.required.phone'), type: 'error' }],
  account: [{ required: true, message: t('pages.login.required.account'), type: 'error' }],
  password: [{ required: true, message: t('pages.login.required.password'), type: 'error' }],
  verifyCode: [{ required: true, message: t('pages.login.required.verification'), type: 'error' }],
};

const type = ref('password');

const form = ref<FormInstanceFunctions>();
const formData = ref({ ...INITIAL_DATA });
const showPsw = ref(false);

const [countDown, handleCounter] = useCounter();

const switchLoginType = (val: string) => {
  type.value = val;
};

const switchToRegister = () => {
  emit('switchType', 'register');
};

const router = useRouter();
const route = useRoute();

/**
 * 发送验证码
 */
const sendCode = () => {
  form.value.validate({ fields: ['phone'] }).then((e) => {
    if (e === true) {
      handleCounter();
    }
  });
};

const onSubmit = async (ctx: SubmitContext) => {
  if (ctx.validateResult === true) {
    try {
      await userStore.login(formData.value);
      await userStore.getUserInfo();

      const redirect = route.query.redirect as string;
      const redirectUrl = redirect ? decodeURIComponent(redirect) : '/dashboard';
      router.push(redirectUrl);
    } catch (e) {
      console.log(e);
      MessagePlugin.error(e.message);
    }
  }
};
</script>
<style lang="less" scoped>
@import '../index.less';

.tip-container { 
  color: rgba(255, 255, 255, 0.7); 
  display: flex; 
  gap: 12px; 
  align-items: center;
  margin-bottom: 20px;
  font-size: 14px;
}

.tip-container .refresh { 
  color: rgba(99, 179, 237, 0.8); 
  display: flex; 
  align-items: center; 
  gap: 6px;
  cursor: pointer;
  transition: color 0.3s ease;
  
  &:hover {
    color: rgba(99, 179, 237, 1);
  }
}

.qrcode-container {
  border-radius: 16px;
  box-shadow: 0 12px 40px rgba(99, 179, 237, 0.15);
  overflow: hidden;
  padding: 20px;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
}

.verification-code {
  display: flex;
  gap: 12px;

  :deep(.t-input) {
    flex: 1;
  }

  :deep(.t-button) {
    border: 1px solid rgba(255, 255, 255, 0.2);
    flex-shrink: 0;
  }
}

.check-container {
  .tip { 
    color: rgba(99, 179, 237, 0.8);
    transition: color 0.3s ease;
    cursor: pointer;
    
    &:hover {
      color: rgba(99, 179, 237, 1);
    }
  }
}

:deep(.t-checkbox) {
  color: rgba(255, 255, 255, 0.6);
  
  .t-checkbox__input {
    background-color: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.2);
  }

  &:hover .t-checkbox__input {
    border-color: rgba(255, 255, 255, 0.3);
  }

  &.t-is-checked .t-checkbox__input {
    background-color: rgba(99, 179, 237, 0.8);
    border-color: rgba(99, 179, 237, 0.8);
  }
}

.icon-btn {
  cursor: pointer;
  transition: all 0.3s ease;
  
  &:hover {
    transform: scale(1.1);
  }
}
</style>