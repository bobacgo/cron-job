<template>
  <div class="user-management">
    <t-card class="list-card-container" :bordered="false">
      <t-row justify="space-between" align="center" class="operation-row">
        <div class="left-operation-container">
          <t-button theme="primary" @click="handleAdd">
            <template #icon><add-icon /></template>
            添加用户
          </t-button>
          <t-button 
            variant="base" 
            theme="danger" 
            :disabled="!selectedRowKeys.length" 
            @click="handleBatchDelete"
          >
            <template #icon><delete-icon /></template>
            批量删除
          </t-button>
          <span v-if="selectedRowKeys.length > 0" class="selected-count">
            已选择 {{ selectedRowKeys.length }} 项
          </span>
        </div>
        <div class="search-input">
          <t-input
            v-model="searchValue"
            placeholder="搜索用户账号、手机号或邮箱"
            clearable
            @clear="handleSearch"
            @press-enter="handleSearch"
          >
            <template #suffix-icon>
              <search-icon size="16px" @click="handleSearch" />
            </template>
          </t-input>
        </div>
      </t-row>

      <div class="table-wrapper">
        <t-table
          :data="userList"
          :columns="columns"
          row-key="id"
          vertical-align="middle"
          :hover="true"
          :pagination="pagination"
          :selected-row-keys="selectedRowKeys"
          :loading="dataLoading"
          @select-change="handleSelectChange"
          @page-change="handlePageChange"
        >
          <template #role_ids="{ row }">
            <t-select 
              :model-value="row.role_ids ? row.role_ids.split(',').map((id: string) => id.trim()) : []" 
              multiple
              size="small"
              placeholder="请选择角色"
              filterable
              @change="(val) => handleRoleChange(row, val)"
              style="min-width: 120px;"
            >
              <t-option 
                v-for="role in roleList" 
                :key="role.id" 
                :value="String(role.id)" 
                :label="role.role_name"
              />
            </t-select>
          </template>
          
          <template #status="{ row }">
            <t-switch 
              :model-value="row.status === 1" 
              @change="(val) => handleStatusChange(row, val)"
            />
          </template>
          
          <template #register_at="{ row }">
            {{ formatTimestamp(row.register_at) }}
          </template>
          
          <template #login_at="{ row }">
            {{ formatTimestamp(row.login_at) }}
          </template>
          
          <template #updated_at="{ row }">
            {{ formatTimestamp(row.updated_at) }}
          </template>
          
          <template #op="{ row }">
            <t-space>
              <t-link theme="primary" hover="color" @click="handleEdit(row)">
                <template #icon><edit-icon /></template>
                编辑
              </t-link>
              <t-link theme="danger" hover="color" @click="handleDelete(row)">
                <template #icon><delete-icon /></template>
                删除
              </t-link>
            </t-space>
          </template>
        </t-table>
      </div>
    </t-card>

    <!-- 添加/编辑用户对话框 -->
    <t-dialog
      v-model:visible="dialogVisible"
      :header="dialogType === 'add' ? '添加用户' : '编辑用户'"
      :width="600"
      :confirm-on-enter="false"
      @confirm="handleDialogConfirm"
      @close="handleDialogClose"
    >
      <t-form
        ref="formRef"
        :data="formData"
        :rules="formRules"
        label-width="100px"
        @submit="handleDialogConfirm"
      >
        <t-form-item label="账号" name="account">
          <t-input v-model="formData.account" placeholder="请输入账号" :disabled="dialogType === 'edit'" />
        </t-form-item>
        <t-form-item label="手机号" name="phone">
          <t-input v-model="formData.phone" placeholder="请输入手机号" />
        </t-form-item>
        <t-form-item label="邮箱" name="email">
          <t-input v-model="formData.email" placeholder="请输入邮箱" />
        </t-form-item>
        <t-form-item v-if="dialogType === 'add'" label="状态" name="status">
          <t-select v-model="formData.status" placeholder="请选择状态">
            <t-option :value="1" label="启用" />
            <t-option :value="2" label="禁用" />
          </t-select>
        </t-form-item>
        <t-form-item v-if="dialogType === 'add'" label="角色" name="role_ids">
          <t-select 
            v-model="formData.role_ids" 
            multiple
            placeholder="请选择角色，可搜索"
            clearable
            filterable
          >
            <t-option 
              v-for="role in roleList" 
              :key="role.id" 
              :value="String(role.id)" 
              :label="role.role_name"
            />
          </t-select>
        </t-form-item>
        <t-form-item v-if="dialogType === 'add'" label="密码" name="password">
          <t-input v-model="formData.password" type="password" placeholder="请输入密码" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <!-- 删除确认对话框 -->
    <t-dialog
      v-model:visible="confirmVisible"
      header="确认删除"
      :body="confirmBody"
      @confirm="onConfirmDelete"
      @close="onCancel"
    />

    <!-- 编辑状态/角色确认对话框 -->
    <t-dialog
      v-model:visible="editingConfirmVisible"
      :header="editingConfirmType === 'status' ? '确认修改状态' : '确认修改角色'"
      :body="getEditingConfirmBody()"
      @confirm="handleEditingConfirm"
      @close="handleEditingCancel"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { 
  SearchIcon, 
  AddIcon, 
  EditIcon, 
  DeleteIcon 
} from 'tdesign-icons-vue-next';
import { 
  MessagePlugin, 
  FormInstanceFunctions, 
  type PrimaryTableCol 
} from 'tdesign-vue-next';
import dayjs from 'dayjs';
import { getUserList, addUser, updateUser, updateUserStatus, updateUserRole, updateUserPassword, deleteUser, type User, type UserAddReq, type UserUpdateReq } from '@/api/mgr/user'
import { getRoleList, type Role } from '@/api/mgr/role'

// 响应式数据
const userList = ref<User[]>([]);
const roleList = ref<Role[]>([]);
const dataLoading = ref(false);
const selectedRowKeys = ref<(string | number)[]>([]);
const searchValue = ref('');

// 分页配置
const pagination = ref({
  defaultPageSize: 10,
  total: 0,
  current: 1,
  showPageSize: true,
  showJumper: true,
  pageSizeOptions: [10, 20, 50, 100]
});

// 对话框控制
const dialogVisible = ref(false);
const dialogType = ref<'add' | 'edit'>('add');
const confirmVisible = ref(false);
const deleteIdx = ref<number | string | null>(null);

// 状态/角色编辑确认对话框
const editingConfirmVisible = ref(false);
const editingConfirmType = ref<'status' | 'role'>('status');
const editingUser = ref<User | null>(null);
const editingValue = ref<any>(null);
const originalValue = ref<any>(null);

// 表单相关
const formRef = ref<FormInstanceFunctions>();
const formData = ref({
  id: 0,
  account: '',
  phone: '',
  email: '',
  status: 1,
  password: '',
  role_ids: [] as string[]
});
const originalData = ref<User | null>(null);

// 表格列定义
const columns: PrimaryTableCol[] = [
  { colKey: 'row-select', type: 'multiple', width: 64, fixed: 'left' },
  { title: '账号', colKey: 'account', width: 120, fixed: 'left', ellipsis: true },
  { title: '手机号', colKey: 'phone', width: 150, ellipsis: true },
  { title: '邮箱', colKey: 'email', width: 180, ellipsis: true },
  { title: '角色', colKey: 'role_ids', width: 200, },
  { title: '状态', colKey: 'status', width: 80, align: 'center' },
  { title: '注册时间', colKey: 'register_at', width: 160, align: 'center' },
  { title: '注册IP', colKey: 'register_ip', width: 120, ellipsis: true },
  { title: '最后登录时间', colKey: 'login_at', width: 160, align: 'center' },
  { title: '登录IP', colKey: 'login_ip', width: 120, ellipsis: true },
  { title: '更新时间', colKey: 'updated_at', width: 160, align: 'center' },
  { title: '操作人', colKey: 'operator', width: 120, align: 'center' },
  { title: '操作', colKey: 'op', width: 120, fixed: 'right', align: 'center' }
];

// 表单验证规则
const formRules = computed(() => ({
  account: [
    { required: true, message: '账号不能为空' },
    { min: 3, max: 20, message: '账号长度必须在3-20个字符之间' }
  ],
  phone: [
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号格式' }
  ],
  password: dialogType.value === 'add'
    ? [
        { required: true, message: '密码不能为空' },
        { min: 6, max: 20, message: '密码长度必须在6-20个字符之间' }
      ]
    : [
        { validator: (val: string) => !val || (val.length >= 6 && val.length <= 20), message: '密码长度必须在6-20个字符之间' }
      ]
}));

// 格式化函数
const formatTimestamp = (timestamp: number): string => {
  return timestamp ? dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss') : '-';
};

const formatStatus = (status: number): string => {
  return status === 1 ? '启用' : '禁用';
};

const normalizeRoleIds = (ids?: string | string[]) => {
  if (!ids) return '';
  const list = Array.isArray(ids) ? ids : ids.split(',');
  return list.map(id => id.trim()).filter(Boolean).join(',');
};

// 获取用户列表 - 使用新的API接口
const fetchUserList = async () => {
  dataLoading.value = true;
  try {
    const response = await getUserList({
      page: pagination.value.current,
      page_size: pagination.value.defaultPageSize,
      keyword: searchValue.value
    });
    
    const { list, total } = response;
    userList.value = list || [];
    pagination.value.total = total || 0;
  } catch (error) {
    console.error('获取用户列表失败:', error);
    MessagePlugin.error('获取用户列表失败');
  } finally {
    dataLoading.value = false;
  }
};

// 获取角色列表
const fetchRoleList = async () => {
  try {
    const response = await getRoleList({
      page: 1,
      page_size: 100
    });
    roleList.value = response.list || [];
  } catch (error) {
    console.error('获取角色列表失败:', error);
  }
};

// 事件处理函数
const handleSelectChange = (value: (string | number)[]) => {
  selectedRowKeys.value = value;
};

const handlePageChange = (pageInfo: any) => {
  pagination.value.current = pageInfo.current;
  pagination.value.defaultPageSize = pageInfo.pageSize;
  fetchUserList();
};

const handleSearch = () => {
  pagination.value.current = 1;
  fetchUserList();
};

// 添加用户
const handleAdd = () => {
  dialogType.value = 'add';
  formData.value = {
    id: 0,
    account: '',
    phone: '',
    email: '',
    status: 1,
    password: '',
    role_ids: []
  };
  originalData.value = null;
  dialogVisible.value = true;
};

// 编辑用户
const handleEdit = (row: User) => {
  dialogType.value = 'edit';
  originalData.value = { ...row };
  formData.value = {
    id: row.id,
    account: row.account,
    phone: row.phone,
    email: row.email,
    status: row.status,
    password: '',
    role_ids: row.role_ids ? row.role_ids.split(',').map(id => id.trim()) : []
  };
  dialogVisible.value = true;
};

// 处理状态变更
const handleStatusChange = (row: User, isActive: any) => {
  originalValue.value = row.status;
  editingValue.value = isActive ? 1 : 2;
  editingUser.value = row;
  editingConfirmType.value = 'status';
  editingConfirmVisible.value = true;
};

// 处理角色变更
const handleRoleChange = (row: User, selectedRoles: any) => {
  originalValue.value = row.role_ids;
  editingValue.value = Array.isArray(selectedRoles) ? selectedRoles.join(',') : '';
  editingUser.value = row;
  editingConfirmType.value = 'role';
  editingConfirmVisible.value = true;
};

// 删除用户
const handleDelete = (row: User) => {
  deleteIdx.value = row.id;
  confirmVisible.value = true;
};

// 批量删除
const handleBatchDelete = () => {
  if (selectedRowKeys.value.length === 0) {
    MessagePlugin.warning('请选择要删除的用户');
    return;
  }
  deleteIdx.value = null; // 标记为批量删除
  confirmVisible.value = true;
};

// 对话框确认
const handleDialogConfirm = async () => {
  try {
    const valid = await formRef.value?.validate();
    if (!valid) return;

    if (dialogType.value === 'add') {
      // 添加用户
      const addData: UserAddReq = {
        account: formData.value.account,
        password: formData.value.password,
        email: formData.value.email,
        phone: formData.value.phone,
        status: formData.value.status,
        role_ids: formData.value.role_ids.length > 0 ? formData.value.role_ids.join(',') : undefined,
      };
      await addUser(addData);
      MessagePlugin.success('添加用户成功');
    } else {
      const tasks: Promise<unknown>[] = [];
      const currentRoleIds = normalizeRoleIds(formData.value.role_ids);
      const originRoleIds = normalizeRoleIds(originalData.value?.role_ids);

      const baseChanged = originalData.value
        ? formData.value.phone !== originalData.value.phone || formData.value.email !== originalData.value.email
        : true;
      if (baseChanged) {
        const updateData: UserUpdateReq = {
          id: formData.value.id,
          email: formData.value.email,
          phone: formData.value.phone,
        };
        tasks.push(updateUser(updateData));
      }

      const statusChanged = originalData.value ? formData.value.status !== originalData.value.status : true;
      if (statusChanged) {
        tasks.push(updateUserStatus({ id: formData.value.id, status: formData.value.status }));
      }

      if (currentRoleIds !== originRoleIds) {
        tasks.push(updateUserRole({ id: formData.value.id, role_ids: currentRoleIds }));
      }

      if (formData.value.password) {
        tasks.push(updateUserPassword({ id: formData.value.id, new_password: formData.value.password }));
      }

      if (!tasks.length) {
        MessagePlugin.info('未修改任何信息');
        dialogVisible.value = false;
        return;
      }

      await Promise.all(tasks);
      MessagePlugin.success('编辑用户成功');
    }

    dialogVisible.value = false;
    formData.value.password = '';
    fetchUserList();
  } catch (error: any) {
    const message = error.response?.data?.message || '操作失败';
    MessagePlugin.error(message);
  }
};

const handleDialogClose = () => {
  formRef.value?.clearValidate();
};

// 删除确认
const confirmBody = computed(() => {
  if (deleteIdx.value === null) {
    return `确定要删除选中的 ${selectedRowKeys.value.length} 个用户吗？`;
  }
  const user = userList.value.find(u => u.id === deleteIdx.value);
  return user ? `确定要删除用户 "${user.account}" 吗？删除后该用户的所有信息将被清空且无法恢复。` : '';
});

const onConfirmDelete = async () => {
  try {
    if (deleteIdx.value === null) {
      // 批量删除
      const ids = selectedRowKeys.value.map(id => Number(id))
      await deleteUser(ids);
      selectedRowKeys.value = [];
      MessagePlugin.success('批量删除成功');
    } else {
      // 单个删除
      const ids = [Number(deleteIdx.value)]
      await deleteUser(ids);
      MessagePlugin.success('删除成功');
    }
    fetchUserList();
  } catch (error: any) {
    const message = error.response?.data?.message || '删除失败';
    MessagePlugin.error(message);
  } finally {
    confirmVisible.value = false;
    deleteIdx.value = null;
  }
};

const onCancel = () => {
  confirmVisible.value = false;
  deleteIdx.value = null;
};

const getEditingConfirmBody = () => {
  if (!editingUser.value) return '';
  if (editingConfirmType.value === 'status') {
    return `确定要将用户 "${editingUser.value.account}" 的状态改为 "${editingValue.value === 1 ? '启用' : '禁用'}" 吗？`;
  }
  if (editingConfirmType.value === 'role') {
    const newRoles = editingValue.value || '无';
    const oldRoles = originalValue.value || '无';
    return `确定要修改用户 "${editingUser.value.account}" 的角色吗？\n原角色：${oldRoles}\n新角色：${newRoles}`;
  }
  return '确定要修改用户角色吗？';
};

const handleEditingConfirm = async () => {
  try {
    if (!editingUser.value) return;
    
    if (editingConfirmType.value === 'status') {
      await updateUserStatus({ 
        id: editingUser.value.id, 
        status: editingValue.value 
      });
      MessagePlugin.success('状态更新成功');
    } else if (editingConfirmType.value === 'role') {
      await updateUserRole({ 
        id: editingUser.value.id, 
        role_ids: editingValue.value 
      });
      MessagePlugin.success('角色更新成功');
    }
    
    editingConfirmVisible.value = false;
    fetchUserList();
  } catch (error: any) {
    const message = error.response?.data?.message || '更新失败';
    MessagePlugin.error(message);
    // 恢复原值
    if (editingUser.value) {
      if (editingConfirmType.value === 'status') {
        editingUser.value.status = originalValue.value;
      } else if (editingConfirmType.value === 'role') {
        editingUser.value.role_ids = originalValue.value;
      }
    }
  }
};

const handleEditingCancel = () => {
  if (editingUser.value) {
    if (editingConfirmType.value === 'status') {
      editingUser.value.status = originalValue.value;
    } else if (editingConfirmType.value === 'role') {
      editingUser.value.role_ids = originalValue.value;
    }
  }
  editingConfirmVisible.value = false;
};

// 生命周期
onMounted(() => {
  fetchRoleList();
  fetchUserList();
});
</script>

<style scoped lang="less">
.user-management {
  padding: 24px;
  background-color: var(--td-bg-color-container);
  min-height: 100%;
}

.list-card-container {
  :deep(.t-card__body) {
    padding: 24px;
  }
}

.operation-row {
  margin-bottom: 16px;

  .left-operation-container {
    display: flex;
    align-items: center;
    gap: 12px;

    .selected-count {
      color: var(--td-text-color-secondary);
      font-size: 14px;
    }
  }

  .search-input {
    width: 360px;
  }
}

.table-wrapper {
  overflow-x: auto;
  overflow-y: hidden;
  border-radius: var(--td-radius-medium);
  
  :deep(.t-table) {
    min-width: 100%;
    
    .t-table__content {
      border-radius: var(--td-radius-medium);
    }
  }
}

.text-secondary {
  color: var(--td-text-color-secondary);
}

:deep(.t-table) {
  .t-table__content {
    border-radius: var(--td-radius-medium);
  }
}

@media screen and (max-width: 768px) {
  .operation-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;

    .search-input {
      width: 100%;
    }
  }
}
</style>