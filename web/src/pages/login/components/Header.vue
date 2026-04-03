<template>
  <header class="login-header">
    <logo-full-icon class="logo" />
    <div class="operations-container">
      <t-button theme="default" shape="square" variant="text" @click="navToGitHub">
        <t-icon name="logo-github" class="icon" />
      </t-button>
      <t-button theme="default" shape="square" variant="text" @click="navToHelper">
        <t-icon name="help-circle" class="icon" />
      </t-button>
      <t-button theme="default" shape="square" variant="text" @click="toggleSettingPanel">
        <t-icon name="setting" class="icon" />
      </t-button>
    </div>
  </header>
</template>
<script setup lang="ts">
import LogoFullIcon from '@/assets/assets-logo-full.svg?component';
import { useSettingStore } from '@/store';

const settingStore = useSettingStore();
const toggleSettingPanel = () => {
  settingStore.updateConfig({
    showSettingPanel: true,
  });
};

const navToGitHub = () => {
  window.open('https://github.com/tencent/tdesign-vue-next-starter');
};

const navToHelper = () => {
  window.open('http://tdesign.tencent.com/starter/docs/get-started');
};
</script>
<style lang="less" scoped>
.login-header {
  padding: 20px 40px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: auto;
  z-index: 4;
  backdrop-filter: blur(20px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);

  .logo {
    width: 140px;
    height: 30px;
    filter: brightness(1.1) drop-shadow(0 4px 12px rgba(99, 179, 237, 0.2));
    transition: filter 0.3s ease;
    
    &:hover {
      filter: brightness(1.2) drop-shadow(0 6px 16px rgba(99, 179, 237, 0.3));
    }
  }

  .operations-container {
    display: flex;
    align-items: center;
    gap: 8px;

    .t-button {
      margin: 0;
      background: rgba(255, 255, 255, 0.05);
      border-radius: 10px;
      border: 1px solid rgba(255, 255, 255, 0.1);
      color: rgba(255, 255, 255, 0.6);
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      padding: 8px;

      .icon {
        color: rgba(255, 255, 255, 0.7);
        font-size: 18px;
      }

      &:hover {
        background: rgba(255, 255, 255, 0.08);
        border-color: rgba(99, 179, 237, 0.4);
        color: rgba(99, 179, 237, 0.9);
        transform: translateY(-2px);
        box-shadow: 0 8px 16px rgba(99, 179, 237, 0.15);

        .icon {
          color: rgba(99, 179, 237, 0.9);
        }
      }

      &:active {
        transform: translateY(0);
      }
    }
  }
}
</style>
