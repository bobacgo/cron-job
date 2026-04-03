<template>
  <div class="role-management">
    <t-card class="list-card-container" :bordered="false">
      <t-row justify="space-between" align="center" class="operation-row">
        <div class="left-operation-container">
          <t-button theme="primary" @click="handleAdd">
            <template #icon><add-icon /></template>
            添加角色
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
            placeholder="搜索角色名称或描述"
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

      <t-table
        :data="roleList"
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
        <template #status="{ row }">
          <t-tag :theme="row.status === 1 ? 'success' : 'danger'" variant="light">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </t-tag>
        </template>

        <template #user_count="{ row }">
          <span>{{ (row.user_count ?? 0) }}</span>
        </template>

        <template #created_at="{ row }">
          {{ formatTimestamp(row.created_at) }}
        </template>

        <template #op="{ row }">
          <t-space>
            <t-link theme="primary" hover="color" @click="handleEdit(row)">
              <template #icon><edit-icon /></template>
              编辑
            </t-link>
            <t-link theme="primary" hover="color" @click="handlePermission(row)">
              <template #icon><setting-icon /></template>
              权限
            </t-link>
            <t-link theme="danger" hover="color" @click="handleDelete(row)">
              <template #icon><delete-icon /></template>
              删除
            </t-link>
          </t-space>
        </template>
      </t-table>
    </t-card>

    <t-dialog v-model:visible="dialogVisible" :header="dialogType==='add'?'添加角色':'编辑角色'" :width="560" @confirm="handleDialogConfirm" @close="handleDialogClose">
      <t-form ref="formRef" :data="formData" :rules="formRules" label-width="100px" @submit="handleDialogConfirm">
        <t-form-item label="角色名称" name="role_name">
          <t-input v-model="formData.role_name" placeholder="例如 admin" />
        </t-form-item>
        <t-form-item label="描述" name="description">
          <t-input v-model="formData.description" placeholder="角色描述" />
        </t-form-item>
        <t-form-item label="状态" name="status">
          <t-select v-model="formData.status" placeholder="请选择状态">
            <t-option :value="1" label="启用" />
            <t-option :value="0" label="禁用" />
          </t-select>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog v-model:visible="confirmVisible" header="确认删除" :body="confirmBody" @confirm="onConfirmDelete" @close="onCancel" />

    <!-- 权限管理弹框 -->
    <t-dialog 
      v-model:visible="permissionDialogVisible" 
      header="权限管理" 
      :width="600"
      @confirm="handlePermissionConfirm"
      @close="handlePermissionClose"
    >
      <div class="permission-dialog">
        <p class="role-name">角色: {{ permissionFormData.role_name }}</p>
        <t-tree
          v-model:expanded="expandedKeys"
          :value="checkedMenuIds"
          :data="menuTreeData"
          checkable
          :check-strictly="false"
          value-mode="all"
          :keys="{ value: 'id', label: 'name', children: 'children' }"
          @change="(val) => checkedMenuIds = val"
        >
          <template #label="{ node }">
            <t-icon v-if="node.data.icon" :name="node.data.icon" style="margin-right: 8px;" />
            {{ node.data.name }}
          </template>
        </t-tree>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { MessagePlugin, type FormInstanceFunctions, type PrimaryTableCol, type FormRules } from 'tdesign-vue-next';
import { AddIcon, DeleteIcon, EditIcon, SearchIcon, SettingIcon } from 'tdesign-icons-vue-next';
import { getRoleList, addRole, updateRole, deleteRole, saveRolePermissions, getRolePermissions } from '@/api/mgr/role';
import { getMenuTree, type MenuItem } from '@/api/mgr/menu';
import type { Role, RoleListReq, RoleCreateReq, RoleUpdateReq } from '@/api/mgr/role';
import dayjs from 'dayjs';

const roleList = ref<Role[]>([]);
const dataLoading = ref(false);
const selectedRowKeys = ref<(string | number)[]>([]);
const searchValue = ref('');

const pagination = ref({ defaultPageSize: 10, total: 0, current: 1, showPageSize: true, showJumper: true, pageSizeOptions: [10,20,50,100] });

const dialogVisible = ref(false);
const dialogType = ref<'add'|'edit'>('add');
const confirmVisible = ref(false);
const deleteIdx = ref<number | string | null>(null);

// 权限管理相关
const permissionDialogVisible = ref(false);
const menuTreeData = ref<MenuItem[]>([]);
const expandedKeys = ref<(string | number)[]>([]);
const checkedMenuIds = ref<(string | number)[]>([]);
const permissionFormData = ref({ id: 0, role_name: '' });

const formRef = ref<FormInstanceFunctions>();
const formData = ref<RoleCreateReq | RoleUpdateReq>({ id: 0 as unknown as number, role_name: '', description: '', status: 1 });

const columns: PrimaryTableCol[] = [
  { colKey: 'row-select', type: 'multiple', width: 64, fixed: 'left' },
  { title: 'ID', colKey: 'id', width: 80 },
  { title: '角色名称', colKey: 'role_name', width: 180 },
  { title: '描述', colKey: 'description', width: 240 },
  { title: '用户数', colKey: 'user_count', width: 100, align: 'center' },
  { title: '状态', colKey: 'status', width: 100, align: 'center' },
  { title: '操作人', colKey: 'operator', width: 120, align: 'center' },
  { title: '创建时间', colKey: 'created_at', width: 180, align: 'center' },
  { title: '操作', colKey: 'op', width: 140, fixed: 'right', align: 'center' }
];

const formRules: FormRules = { role_name: [{ required: true, message: '角色名称不能为空' }], description: [] };

const formatTimestamp = (timestamp: number): string => {
  return timestamp ? dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss') : '-';
};

const fetchRoleList = async () => {
  dataLoading.value = true;
  try {
    const resp = await getRoleList({ page: pagination.value.current, page_size: pagination.value.defaultPageSize, role_name: searchValue.value });
    roleList.value = resp.list || [];
    pagination.value.total = resp.total || 0;
  } catch (e) { MessagePlugin.error('获取角色列表失败'); console.error(e); } finally { dataLoading.value = false; }
};

const handleSelectChange = (v: (string|number)[]) => selectedRowKeys.value = v;
const handlePageChange = (pageInfo: { current: number; pageSize: number }) => { pagination.value.current = pageInfo.current; pagination.value.defaultPageSize = pageInfo.pageSize; fetchRoleList(); };
const handleSearch = () => { pagination.value.current = 1; fetchRoleList(); };
const handleAdd = () => { dialogType.value = 'add'; formData.value = { id:0, role_name:'', description:'', status:1 }; dialogVisible.value = true; };
const handleEdit = (row: Role) => { dialogType.value='edit'; formData.value = { ...(row as RoleUpdateReq), id: row.id }; dialogVisible.value = true; };
const handleDelete = (row: Role) => { deleteIdx.value = row.id; confirmVisible.value = true; };
const handleBatchDelete = () => { if(!selectedRowKeys.value.length){ MessagePlugin.warning('请选择要删除的角色'); return; } deleteIdx.value = null; confirmVisible.value = true; };

const handleDialogConfirm = async () => {
  try {
    const valid = await formRef.value?.validate();
    if (!valid) return;
    if (dialogType.value === 'add') { 
      const createReq = formData.value as RoleCreateReq;
      await addRole(createReq); 
      MessagePlugin.success('添加角色成功'); 
    }
    else { 
      const updateReq = formData.value as RoleUpdateReq;
      await updateRole(updateReq); 
      MessagePlugin.success('编辑角色成功'); 
    }
    dialogVisible.value = false; fetchRoleList();
  } catch (e:any) { const msg = e.response?.data?.message || '操作失败'; MessagePlugin.error(msg); }
};

const handleDialogClose = () => formRef.value?.clearValidate();

const confirmBody = computed(() => { if (deleteIdx.value === null) return `确定要删除选中的 ${selectedRowKeys.value.length} 个角色吗？`; const r = roleList.value.find(i=>i.id===deleteIdx.value); return r?`确定要删除角色 "${r.role_name}" 吗？` : ''; });

const onConfirmDelete = async () => {
  try {
    if (deleteIdx.value === null) { const ids = selectedRowKeys.value.map(id=>Number(id)); await deleteRole(ids); selectedRowKeys.value = []; MessagePlugin.success('批量删除成功'); }
    else { const ids=[Number(deleteIdx.value)]; await deleteRole(ids); MessagePlugin.success('删除成功'); }
    fetchRoleList();
  } catch (e:any) { const msg = e.response?.data?.message || '删除失败'; MessagePlugin.error(msg); } finally { confirmVisible.value=false; deleteIdx.value=null; }
};

const onCancel = () => { confirmVisible.value=false; deleteIdx.value=null; };

// 获取所有叶子节点ID（用于过滤父节点）
const getLeafNodeIds = (nodes: MenuItem[]): number[] => {
  const leafIds: number[] = [];
  const traverse = (items: MenuItem[]) => {
    items.forEach(item => {
      if (!item.children || item.children.length === 0) {
        leafIds.push(item.id);
      } else {
        traverse(item.children);
      }
    });
  };
  traverse(nodes);
  return leafIds;
};

// 权限管理相关函数
const handlePermission = async (row: Role) => {
  try {
    permissionFormData.value = { id: row.id, role_name: row.role_name };
    
    // 获取菜单树
    const menus = await getMenuTree();
    menuTreeData.value = Array.isArray(menus) ? menus : (menus as any).list || [];
    
    // 获取角色已有权限
    const res = await getRolePermissions(row.id);
    
    // 重置状态
    checkedMenuIds.value = [];
    expandedKeys.value = [];
    
    if (res && res.menu_ids && res.menu_ids.length > 0) {
      // 获取所有叶子节点ID
      const leafIds = getLeafNodeIds(menuTreeData.value);
      // 只保留叶子节点，过滤掉父节点
      checkedMenuIds.value = res.menu_ids.filter(id => leafIds.includes(id));
    }
    
    // 不自动展开树，保持默认收起状态
    // 最后打开弹窗
    permissionDialogVisible.value = true;
  } catch (e) {
    MessagePlugin.error('获取权限数据失败');
    console.error(e);
  }
};

const handlePermissionConfirm = async () => {
  try {
    const menuIds = checkedMenuIds.value.map(id => Number(id));
    await saveRolePermissions(permissionFormData.value.id, menuIds);
    MessagePlugin.success('权限设置成功');
    permissionDialogVisible.value = false;
  } catch (e: any) {
    const msg = e.response?.data?.message || '权限设置失败';
    MessagePlugin.error(msg);
  }
};

const handlePermissionClose = () => {
  permissionFormData.value = { id: 0, role_name: '' };
  checkedMenuIds.value = [];
  expandedKeys.value = [];
};

onMounted(()=>{ fetchRoleList(); });
</script>

<style scoped lang="less">
.role-management { padding:24px; background-color: var(--td-bg-color-container); min-height:100%; }
.list-card-container { :deep(.t-card__body){ padding:24px; } }
.operation-row { margin-bottom:16px; .left-operation-container{ display:flex; align-items:center; gap:12px; .selected-count{ color: var(--td-text-color-secondary); font-size:14px; } } .search-input{ width:360px; } }
:deep(.t-table){ .t-table__content{ border-radius: var(--td-radius-medium); } }

.permission-dialog {
  padding: 16px 0;
  
  .role-name {
    margin-bottom: 16px;
    font-weight: 500;
    color: var(--td-text-color-primary);
  }
  
  :deep(.t-tree) {
    max-height: 400px;
    overflow-y: auto;
  }
}

@media screen and (max-width:768px){ .operation-row{ flex-direction:column; align-items:flex-start; gap:12px; .search-input{ width:100%; } } }
</style>
