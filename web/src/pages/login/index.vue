<template>
  <div class="login-wrapper">
    <!-- macOS 风格动态壁纸背景 -->
    <div class="wallpaper-container">
      <div class="wallpaper-bg"></div>
      <div class="wallpaper-overlay"></div>
    </div>

    <!-- 主内容区 -->
    <div class="login-content">
      <!-- 顶部状态栏 -->
      <div class="top-status-bar">
        <div class="right-info">
          <span class="icon">🔋</span>
          <span class="icon">📶</span>
          <span class="icon">Control Center</span>
        </div>
      </div>

      <!-- 锁屏大时钟 -->
      <div class="lock-screen-clock">
        <div class="date">{{ currentDate }}</div>
        <div class="time">{{ currentTime }}</div>
      </div>

      <!-- 底部登录区 -->
      <div class="bottom-login-container">
        <!-- 用户头像 -->
        <div class="avatar-section">
          <div class="avatar-box">
            <img src="https://tdesign.gtimg.com/site/avatar.jpg" alt="User Avatar" class="avatar-img" />
          </div>
          <h2 class="account-name">{{ selectedAccount }}</h2>
        </div>

        <!-- 登录表单 -->
        <div class="login-form-container">
          <login v-if="type === 'login'" @switch-type="switchType" />
          <register v-else @register-success="switchType('login')" @switch-type="switchType" />
        </div>

        <!-- 底部按钮 -->
        <div class="bottom-actions">
          <div class="action-btn" title="Sleep">
            <span class="icon">🌙</span>
            <span>Sleep</span>
          </div>
          <div class="action-btn" title="Restart">
            <span class="icon">🔄</span>
            <span>Restart</span>
          </div>
          <div class="action-btn" title="Shut Down">
            <span class="icon">⭕</span>
            <span>Shut Down</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import Login from './components/Login.vue';
import Register from './components/Register.vue';

defineOptions({
  name: 'LoginIndex',
});

const type = ref('login');
const selectedAccount = ref('Admin');
const currentTime = ref('');
const currentDate = ref('');

const switchType = (val: string) => {
  type.value = val;
};

// 更新时间
onMounted(() => {
  const updateTime = () => {
    const now = new Date();
    currentTime.value = now.toLocaleTimeString('en-US', { 
      hour: 'numeric', 
      minute: '2-digit',
      hour12: false 
    });
    currentDate.value = now.toLocaleDateString('zh-CN', { 
      weekday: 'long', 
      month: 'long', 
      day: 'numeric' 
    });
  };
  updateTime();
  setInterval(updateTime, 1000);
});
</script>

<style lang="less" scoped>
@import './index.less';
</style>
