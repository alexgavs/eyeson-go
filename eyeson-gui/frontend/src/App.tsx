/**
 * Copyright (c) 2026 Alexander G.
 * Author: Alexander G. (Samsonix)
 * License: MIT
 * Project: EyesOn SIM Management System
 */

import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import {
  Login,
  GetSims,
  GetStats,
  UpdateSim,
  ChangeStatus,
  GetJobStatus,
  GetJobs,
  GetUsers,
  CreateUser,
  UpdateUser,
  DeleteUser,
  ResetUserPassword,
  GetRoles,
  GetAPIStatus,
  GetUpstream,
  SetUpstream,
  QueueTask,
  GetManualSyncStatus,
  TriggerManualFullSync,
  ManualSyncStatus,
  APIStatusResponse,
  User,
  Role
} from './api';

import type { NavPage, PendingJob, PendingStatus, Toast } from './types/app';
import { ALL_COLUMNS, STORAGE_KEYS } from './constants/app';
import { CookieManager } from './utils/cookies';
import { SessionManager } from './utils/session';
import { formatDate } from './utils/format';

import { QueueView } from './components/QueueView';
import { ToastContainer } from './components/ToastContainer';
import { SimDetailModal } from './components/SimDetailModal';
import { JobStatusBadge, StatusBadge } from './components/StatusBadges';

// ==================== КОНСТАНТЫ ====================

// ==================== ГЛАВНЫЙ КОМПОНЕНТ ====================

function App() {
  // Авторизация
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [isCheckingSession, setIsCheckingSession] = useState(true);
  
  // Навигация
  const [navPage, setNavPage] = useState<NavPage>('sims');
  
  // SIM данные
  const [allSimsData, setAllSimsData] = useState<any[]>([]); // Все SIM для статистики
  const [sims, setSims] = useState<any[]>([]);  // Текущая страница
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  
  // Stats
  const [serverStats, setServerStats] = useState<any>(null);
  const [statsLoaded, setStatsLoaded] = useState(false);
  
  // Фильтры и сортировка
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [pagination, setPagination] = useState({ start: 0, limit: 500 }); // Увеличено до 500 для локальной БД
  const [sort, setSort] = useState({ by: "", direction: "ASC" });
  
  // Выбор
  const [selectedSims, setSelectedSims] = useState<Set<string>>(new Set());
  const [selectedSim, setSelectedSim] = useState<any>(null);
  
  // Редактирование
  const [editMode, setEditMode] = useState(false);
  const [editValues, setEditValues] = useState<any>({});

  // Pending статусы (legacy - для отображения спиннеров)
  const [pendingStatuses, setPendingStatuses] = useState<Map<string, PendingStatus>>(new Map());
  
  // Pending Jobs (новый подход - polling по Job ID)
  const [pendingJobs, setPendingJobs] = useState<PendingJob[]>([]);
  
  // Jobs - загружаем все и делаем клиентскую пагинацию
  const [allJobs, setAllJobs] = useState<any[]>([]);  // Все jobs с сервера
  const [jobs, setJobs] = useState<any[]>([]);        // Текущая страница
  const [jobsPage, setJobsPage] = useState(1);        // Текущая страница
  const [jobsPerPage, setJobsPerPage] = useState(25); // Записей на страницу
  const [jobFilters, setJobFilters] = useState({ jobId: '', status: '' });
  const [jobsLoaded, setJobsLoaded] = useState(false); // Загружены ли данные
  
  // User Management
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [usersLoaded, setUsersLoaded] = useState(false);
  const [showUserModal, setShowUserModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [userForm, setUserForm] = useState({ username: '', email: '', password: '', role: 'Viewer' });
  const [showResetPasswordModal, setShowResetPasswordModal] = useState(false);
  const [resetPasswordUserId, setResetPasswordUserId] = useState<number | null>(null);
  const [newPassword, setNewPassword] = useState('');

  // API Status (Admin only)
  const [apiStatus, setApiStatus] = useState<APIStatusResponse | null>(null);
  const [apiStatusLoading, setApiStatusLoading] = useState(false);
  const [isAdmin, setIsAdmin] = useState(false);

  // Upstream selection (Admin only, persisted on server; applies after restart)
  const [upstreamCfg, setUpstreamCfg] = useState<any>(null);
  const [upstreamLoading, setUpstreamLoading] = useState(false);
  const [upstreamSaving, setUpstreamSaving] = useState(false);
  const [upstreamSelectedDraft, setUpstreamSelectedDraft] = useState<'pelephone' | 'simulator'>('pelephone');
  const [upstreamRestartRequired, setUpstreamRestartRequired] = useState(false);

  // Manual sync (Admin only) - always from Pelephone
  const [manualSyncStatus, setManualSyncStatus] = useState<ManualSyncStatus | null>(null);
  const [manualSyncLoading, setManualSyncLoading] = useState(false);
  const [manualSyncStarting, setManualSyncStarting] = useState(false);

  // Theme
  const [theme, setTheme] = useState<'dark' | 'light'>(() => {
    const saved = localStorage.getItem(STORAGE_KEYS.theme);
    return (saved === 'light' || saved === 'dark') ? saved : 'dark';
  });

  // Modal dragging state
  const [modalPosition, setModalPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  // Modal drag handlers
  const handleMouseDown = (e: React.MouseEvent) => {
    if ((e.target as HTMLElement).closest('.btn-close')) return;
    setIsDragging(true);
    setDragStart({ x: e.clientX - modalPosition.x, y: e.clientY - modalPosition.y });
  };

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isDragging) return;
    setModalPosition({ x: e.clientX - dragStart.x, y: e.clientY - dragStart.y });
  }, [isDragging, dragStart]);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    }
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, handleMouseMove, handleMouseUp]);

  // Reset modal position when opening
  const resetModalPosition = () => setModalPosition({ x: 0, y: 0 });

  // Колонки - порядок и видимость из cookies
  const [columnOrder, setColumnOrder] = useState<string[]>(() => {
    try {
      // Сначала пробуем cookies
      const cookieVal = CookieManager.get(STORAGE_KEYS.columnOrder);
      if (cookieVal) {
        const parsed = JSON.parse(cookieVal);
        // Убедимся, что все ключи существуют
        if (Array.isArray(parsed) && parsed.every(k => ALL_COLUMNS[k])) {
          return parsed;
        }
      }
      // Fallback на localStorage
      const saved = localStorage.getItem(STORAGE_KEYS.columnOrder);
      if (saved) {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed) && parsed.every(k => ALL_COLUMNS[k])) {
          return parsed;
        }
      }
    } catch {}
    return Object.keys(ALL_COLUMNS);
  });

  const [visibleColumns, setVisibleColumns] = useState<string[]>(() => {
    try {
      // Сначала пробуем cookies
      const cookieVal = CookieManager.get(STORAGE_KEYS.columns);
      if (cookieVal) {
        const parsed = JSON.parse(cookieVal);
        if (Array.isArray(parsed) && parsed.every(k => ALL_COLUMNS[k])) {
          return parsed;
        }
      }
      // Fallback на localStorage
      const saved = localStorage.getItem(STORAGE_KEYS.columns);
      if (saved) {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed) && parsed.every(k => ALL_COLUMNS[k])) {
          return parsed;
        }
      }
    } catch {}
    return Object.keys(ALL_COLUMNS).filter(k => ALL_COLUMNS[k].default);
  });
  
  // Dragging state для перестановки колонок
  const [draggedColumn, setDraggedColumn] = useState<string | null>(null);
  
  // Меню колонок
  const [showColumnMenu, setShowColumnMenu] = useState(false);
  const [columnMenuPos, setColumnMenuPos] = useState({ x: 0, y: 0 });
  const columnMenuRef = useRef<HTMLDivElement>(null);
  
  // Toast
  const [toasts, setToasts] = useState<Toast[]>([]);
  const toastIdRef = useRef(0);
  
  // Refs
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // ==================== ЭФФЕКТЫ ====================

  // Применение темы
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    document.body.setAttribute('data-theme', theme);
    localStorage.setItem(STORAGE_KEYS.theme, theme);
  }, [theme]);

  // Проверка сохраненной сессии при загрузке
  useEffect(() => {
    const checkSession = async () => {
      const session = SessionManager.load();
      if (session) {
        console.log('[Session] Found saved session, restoring...');
        setUsername(session.username);
        setIsLoggedIn(true);
      }
      setIsCheckingSession(false);
    };
    checkSession();
  }, []);

  // Сохранение колонок в cookies и localStorage
  useEffect(() => {
    const value = JSON.stringify(visibleColumns);
    localStorage.setItem(STORAGE_KEYS.columns, value);
    CookieManager.set(STORAGE_KEYS.columns, value);
  }, [visibleColumns]);

  // Сохранение порядка колонок в cookies и localStorage
  useEffect(() => {
    const value = JSON.stringify(columnOrder);
    localStorage.setItem(STORAGE_KEYS.columnOrder, value);
    CookieManager.set(STORAGE_KEYS.columnOrder, value);
  }, [columnOrder]);

  // Закрытие меню колонок при клике вне
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (columnMenuRef.current && !columnMenuRef.current.contains(e.target as Node)) {
        setShowColumnMenu(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Загрузка данных после логина
  useEffect(() => {
    if (isLoggedIn && !isCheckingSession) {
      // Сначала загружаем текущую страницу SIM данных
      loadData(0, pagination.limit).then(() => {
        // Через 1 секунду загружаем статистику
        setTimeout(() => loadStats(false), 1000);
      });
    }
  }, [isLoggedIn, isCheckingSession]);

  // Polling для проверки статусов по Job ID (оптимизированный)
  useEffect(() => {
    if (pendingJobs.length === 0) {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
      return;
    }

    pollingRef.current = setInterval(async () => {
      const updatedJobs: PendingJob[] = [];
      let needsRefresh = false;

      for (const job of pendingJobs) {
        if (Date.now() - job.startTime < 2000) {
          updatedJobs.push(job);
          continue;
        }

        try {
          // Один запрос для проверки всего Job вместо отдельных запросов для каждого MSISDN
          const jobData = await GetJobStatus(job.requestId);
          console.log(`[JobPoll ${job.attempts}/10] Job ${job.requestId}: status=${jobData?.jobStatus}`);

          if (jobData) {
            const jobStatus = jobData.jobStatus;

            if (jobStatus === 'COMPLETED' || jobStatus === 'SUCCESS') {
              console.log(`✓ Job ${job.requestId} completed successfully`);
              showToast(`✓ ${job.msisdns.length} SIM: статус изменён на ${job.targetStatus}`, 'success');
              
              // Обновляем UI для всех SIM в этом Job
              setSims(prev => prev.map(s => 
                job.msisdns.includes(s.MSISDN) ? { ...s, SIM_STATUS_CHANGE: job.targetStatus, _pending: false } : s
              ));
              setSelectedSim((prev: any) => 
                prev && job.msisdns.includes(prev.MSISDN) ? { ...prev, SIM_STATUS_CHANGE: job.targetStatus, _pending: false } : prev
              );
              
              // Убираем из pendingStatuses
              setPendingStatuses(prev => {
                const newMap = new Map(prev);
                job.msisdns.forEach(m => newMap.delete(m));
                return newMap;
              });
              
              needsRefresh = true;
            } else if (jobStatus === 'PARTIAL_SUCCESS') {
              console.log(`⚠ Job ${job.requestId} partially succeeded`);
              showToast(`⚠ Частичный успех: некоторые SIM не обновлены`, 'warning');
              
              setSims(prev => prev.map(s => 
                job.msisdns.includes(s.MSISDN) ? { ...s, _pending: false } : s
              ));
              setPendingStatuses(prev => {
                const newMap = new Map(prev);
                job.msisdns.forEach(m => newMap.delete(m));
                return newMap;
              });
              
              needsRefresh = true;
            } else if (jobStatus === 'FAILED') {
              console.warn(`✗ Job ${job.requestId} failed`);
              showToast(`✗ Ошибка изменения статуса`, 'danger');
              
              setSims(prev => prev.map(s => 
                job.msisdns.includes(s.MSISDN) ? { ...s, _pending: false } : s
              ));
              setPendingStatuses(prev => {
                const newMap = new Map(prev);
                job.msisdns.forEach(m => newMap.delete(m));
                return newMap;
              });
            } else if (job.attempts >= 10) {
              console.warn(`✗ Job ${job.requestId} polling timeout`);
              showToast(`⚠ Таймаут ожидания подтверждения`, 'warning');
              
              setSims(prev => prev.map(s => 
                job.msisdns.includes(s.MSISDN) ? { ...s, _pending: false } : s
              ));
              setPendingStatuses(prev => {
                const newMap = new Map(prev);
                job.msisdns.forEach(m => newMap.delete(m));
                return newMap;
              });
            } else {
              // Job ещё в процессе - продолжаем polling
              updatedJobs.push({ ...job, attempts: job.attempts + 1 });
            }
          } else {
            // Job не найден - увеличиваем счётчик попыток
            if (job.attempts >= 10) {
              showToast(`⚠ Job ${job.requestId} не найден`, 'warning');
              setSims(prev => prev.map(s => 
                job.msisdns.includes(s.MSISDN) ? { ...s, _pending: false } : s
              ));
              setPendingStatuses(prev => {
                const newMap = new Map(prev);
                job.msisdns.forEach(m => newMap.delete(m));
                return newMap;
              });
            } else {
              updatedJobs.push({ ...job, attempts: job.attempts + 1 });
            }
          }
        } catch (e) {
          console.error(`Error polling job ${job.requestId}:`, e);
          updatedJobs.push({ ...job, attempts: job.attempts + 1 });
        }
      }

      setPendingJobs(updatedJobs);
      if (needsRefresh) {
        // Перезагружаем данные и статистику
        console.log('[Polling] Task completed, refreshing SIM data...');
        setTimeout(() => {
          loadData(pagination.start, pagination.limit);
          loadStats(true);
        }, 1000);
      }
    }, 3000); // Polling каждые 3 секунды

    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, [pendingJobs]);

  // ==================== ФУНКЦИИ ====================

  const showToast = useCallback((message: string, type: Toast['type'] = 'info') => {
    const id = ++toastIdRef.current;
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
    }, 5000);
  }, []);

  // Загрузка статистики с сервера (на замену клиентскому расчету)
  const loadStats = useCallback(async (force = false) => {
    try {
      const data = await GetStats(force);
      if (data) {
        setServerStats(data);
        setStatsLoaded(true);
      }
    } catch (e) {
      console.error('Failed to load stats:', e);
    }
  }, []);

  // Use server stats instead of client-side calculation
  const stats = useMemo(() => {
    if (!serverStats) return null;
    return {
      total: serverStats.total || 0,
      by_status: serverStats.by_status || {},
      by_rate_plan: serverStats.by_rate_plan || {},
      active_sessions: serverStats.active_sessions || 0,
      last_updated: serverStats.last_updated
    };
  }, [serverStats]);

  const handleLogin = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    setLoading(true);
    try {
      const result = await Login(username, password);
      if (result === "SUCCESS") {
        // Сохраняем сессию
        SessionManager.save('logged_in', username);
        setIsLoggedIn(true);
        showToast('Успешный вход в систему', 'success');
      } else {
        showToast("Ошибка входа: " + result, 'danger');
      }
    } catch (e) {
      showToast("Ошибка: " + e, 'danger');
    }
    setLoading(false);
  };

  const handleLogout = () => {
    SessionManager.clear();
    setIsLoggedIn(false);
    setUsername("");
    setPassword("");
    setSims([]);
    setAllSimsData([]);
    setStatsLoaded(false);
    showToast('Вы вышли из системы', 'info');
  };

  const loadData = async (start: number, limit: number, searchTerm: string = search, sortBy: string = sort.by, sortDirection: string = sort.direction, status: string = statusFilter) => {
    setLoading(true);
    try {
      const response = await GetSims(searchTerm, start, limit, sortBy, sortDirection, status);
      setSims(response.data || []);
      setTotal(response.count || 0);
      setPagination({ start, limit });
      setSort({ by: sortBy, direction: sortDirection });
      setSelectedSims(new Set());
    } catch (e) {
      showToast("Ошибка загрузки: " + e, 'danger');
    }
    setLoading(false);
  };

  const loadJobs = useCallback(async (forceReload: boolean = false) => {
    // Загружаем только если ещё не загружено или принудительно
    if (jobsLoaded && !forceReload) return;
    
    setLoading(true);
    try {
      const jobId = jobFilters.jobId ? parseInt(jobFilters.jobId) : undefined;
      // Загружаем все записи (limit=500)
      const response = await GetJobs(1, 500, jobId, jobFilters.status || undefined);
      if (response.success) {
        setAllJobs(response.data || []);
        setJobsLoaded(true);
        setJobsPage(1); // Сброс на первую страницу
      }
    } catch (e) {
      showToast("Ошибка загрузки jobs: " + e, 'danger');
    }
    setLoading(false);
  }, [jobFilters, showToast, jobsLoaded]);

  // Клиентская пагинация jobs
  useEffect(() => {
    const startIdx = (jobsPage - 1) * jobsPerPage;
    const endIdx = startIdx + jobsPerPage;
    setJobs(allJobs.slice(startIdx, endIdx));
  }, [allJobs, jobsPage, jobsPerPage]);

  // Общее количество страниц
  const jobsTotalPages = Math.ceil(allJobs.length / jobsPerPage) || 1;

  // Загрузка jobs при переключении на страницу
  useEffect(() => {
    if (isLoggedIn && navPage === 'jobs' && !jobsLoaded) {
      loadJobs();
    }
  }, [isLoggedIn, navPage, jobsLoaded, loadJobs]);

  // ==================== USER MANAGEMENT ====================
  
  const loadUsers = useCallback(async () => {
    try {
      const [usersData, rolesData] = await Promise.all([GetUsers(), GetRoles()]);
      setUsers(usersData);
      setRoles(rolesData);
      setUsersLoaded(true);
    } catch (e) {
      showToast("Ошибка загрузки пользователей: " + e, 'danger');
    }
  }, [showToast]);

  // Загрузка пользователей при переключении на страницу Admin
  useEffect(() => {
    if (isLoggedIn && navPage === 'admin' && !usersLoaded) {
      loadUsers();
    }
  }, [isLoggedIn, navPage, usersLoaded, loadUsers]);

  // ==================== API STATUS (Admin only) ====================
  
  const loadAPIStatus = useCallback(async () => {
    if (!isAdmin) return;
    setApiStatusLoading(true);
    try {
      const status = await GetAPIStatus();
      setApiStatus(status);
    } catch (e) {
      console.error('Error loading API status:', e);
    } finally {
      setApiStatusLoading(false);
    }
  }, [isAdmin]);

  const loadUpstream = useCallback(async () => {
    if (!isAdmin) return;
    setUpstreamLoading(true);
    try {
      const cfg = await GetUpstream();
      if (cfg) {
        setUpstreamCfg(cfg);
        setUpstreamSelectedDraft(cfg.selected);
        setUpstreamRestartRequired(false);
      }
    } catch (e) {
      console.error('Error loading upstream:', e);
    } finally {
      setUpstreamLoading(false);
    }
  }, [isAdmin]);

  const loadManualSyncStatus = useCallback(async () => {
    if (!isAdmin) return;
    setManualSyncLoading(true);
    try {
      const st = await GetManualSyncStatus();
      setManualSyncStatus(st);
      // Обновляем статистику во время синхронизации
      if (st?.running) {
        loadStats(true);
      }
    } catch (e) {
      console.error('Error loading manual sync status:', e);
    } finally {
      setManualSyncLoading(false);
    }
  }, [isAdmin, loadStats]);

  // Предыдущий статус синхронизации для отслеживания завершения
  const prevSyncRunningRef = useRef(false);
  
  useEffect(() => {
    if (!isAdmin) {
      setManualSyncStatus(null);
      return;
    }

    loadManualSyncStatus();
    const t = setInterval(() => {
      loadManualSyncStatus();
    }, 2000); // Уменьшил интервал для более частого обновления
    return () => clearInterval(t);
  }, [isAdmin, loadManualSyncStatus]);
  
  // Обновляем статистику и данные когда синхронизация завершилась
  useEffect(() => {
    const wasRunning = prevSyncRunningRef.current;
    const nowRunning = manualSyncStatus?.running ?? false;
    prevSyncRunningRef.current = nowRunning;
    
    // Если синхронизация только что завершилась
    if (wasRunning && !nowRunning) {
      loadStats(true);
      loadData(pagination.start, pagination.limit);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [manualSyncStatus?.running, loadStats, pagination.start, pagination.limit]);

  // Определяем роль пользователя при входе
  useEffect(() => {
    if (isLoggedIn) {
      // Пробуем загрузить API статус - если успешно, значит админ
      GetAPIStatus().then(status => {
        if (status) {
          setIsAdmin(true);
          setApiStatus(status);
        }
      }).catch(() => {
        setIsAdmin(false);
      });
    } else {
      setIsAdmin(false);
      setApiStatus(null);
    }
  }, [isLoggedIn]);

  useEffect(() => {
    if (isAdmin) {
      loadUpstream();
    } else {
      setUpstreamCfg(null);
      setUpstreamRestartRequired(false);
    }
  }, [isAdmin, loadUpstream]);

  const openCreateUserModal = () => {
    setEditingUser(null);
    setUserForm({ username: '', email: '', password: '', role: 'Viewer' });
    resetModalPosition();
    setShowUserModal(true);
  };

  const openEditUserModal = (user: User) => {
    setEditingUser(user);
    setUserForm({ username: user.username, email: user.email, password: '', role: user.role });
    resetModalPosition();
    setShowUserModal(true);
  };

  const handleSaveUser = async () => {
    if (!userForm.username || !userForm.email) {
      showToast('Заполните все обязательные поля', 'warning');
      return;
    }

    setLoading(true);
    try {
      if (editingUser) {
        // Обновление существующего пользователя
        const result = await UpdateUser(editingUser.id, {
          username: userForm.username,
          email: userForm.email,
          role: userForm.role
        });
        if (result.success) {
          showToast('Пользователь обновлён', 'success');
          setShowUserModal(false);
          loadUsers();
        } else {
          showToast(result.error || 'Ошибка обновления', 'danger');
        }
      } else {
        // Создание нового пользователя
        if (!userForm.password) {
          showToast('Укажите пароль для нового пользователя', 'warning');
          setLoading(false);
          return;
        }
        const result = await CreateUser(userForm);
        if (result.success) {
          showToast('Пользователь создан', 'success');
          setShowUserModal(false);
          loadUsers();
        } else {
          showToast(result.error || 'Ошибка создания', 'danger');
        }
      }
    } catch (e) {
      showToast('Ошибка: ' + e, 'danger');
    }
    setLoading(false);
  };

  const handleDeleteUser = async (user: User) => {
    if (!window.confirm(`Удалить пользователя "${user.username}"?`)) return;
    
    setLoading(true);
    const result = await DeleteUser(user.id);
    if (result.success) {
      showToast('Пользователь удалён', 'success');
      loadUsers();
    } else {
      showToast(result.error || 'Ошибка удаления', 'danger');
    }
    setLoading(false);
  };

  const handleToggleUserActive = async (user: User) => {
    setLoading(true);
    const result = await UpdateUser(user.id, { is_active: !user.is_active });
    if (result.success) {
      showToast(user.is_active ? 'Пользователь деактивирован' : 'Пользователь активирован', 'success');
      loadUsers();
    } else {
      showToast(result.error || 'Ошибка', 'danger');
    }
    setLoading(false);
  };

  const openResetPasswordModal = (userId: number) => {
    setResetPasswordUserId(userId);
    setNewPassword('');
    setShowResetPasswordModal(true);
  };

  const handleResetPassword = async () => {
    if (!newPassword || newPassword.length < 6) {
      showToast('Пароль должен быть минимум 6 символов', 'warning');
      return;
    }
    if (resetPasswordUserId === null) return;

    setLoading(true);
    const result = await ResetUserPassword(resetPasswordUserId, newPassword);
    if (result.success) {
      showToast('Пароль сброшен', 'success');
      setShowResetPasswordModal(false);
    } else {
      showToast(result.error || 'Ошибка сброса пароля', 'danger');
    }
    setLoading(false);
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    loadData(0, pagination.limit, search);
  };

  const handleSort = (column: string) => {
    const config = ALL_COLUMNS[column];
    if (!config?.sortable) return;
    
    if (column !== "CLI" && column !== "MSISDN") {
      showToast('Сортировка по этому полю на сервере недоступна', 'warning');
      return;
    }
    
    let direction = "ASC";
    if (sort.by === column && sort.direction === "ASC") direction = "DESC";
    loadData(0, pagination.limit, search, column, direction);
  };

  const handlePageChange = (newStart: number) => {
    if (newStart >= 0 && newStart < total) {
      loadData(newStart, pagination.limit);
    }
  };

  const handleLimitChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newLimit = parseInt(e.target.value);
    loadData(0, newLimit);
  };

  const toggleSelectAll = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.checked) {
      setSelectedSims(new Set(sims.map(s => s.MSISDN)));
    } else {
      setSelectedSims(new Set());
    }
  };

  const toggleSelect = (msisdn: string) => {
    const newSet = new Set(selectedSims);
    if (newSet.has(msisdn)) newSet.delete(msisdn);
    else newSet.add(msisdn);
    setSelectedSims(newSet);
  };

  const updateStatusOptimistic = (msisdns: string[], targetStatus: string) => {
    const now = Date.now();
    const newPending = new Map(pendingStatuses);

    msisdns.forEach(msisdn => {
      newPending.set(msisdn, { msisdn, targetStatus, attempts: 0, startTime: now });
    });

    setPendingStatuses(newPending);
    setSims(prev => prev.map(sim => {
      if (msisdns.includes(sim.MSISDN)) {
        return { ...sim, SIM_STATUS_CHANGE: targetStatus, _pending: true };
      }
      return sim;
    }));
  };

  const handleBulkStatus = async (status: string) => {
    console.log('[handleBulkStatus] Called with status:', status, 'selectedSims:', selectedSims.size);
    
    if (selectedSims.size === 0) {
      console.log('[handleBulkStatus] No SIMs selected, returning');
      return;
    }
    
    if (!confirm(`Изменить статус ${selectedSims.size} SIM на ${status}?`)) {
      console.log('[handleBulkStatus] User cancelled');
      return;
    }
    
    const msisdns = Array.from(selectedSims);
    console.log('[handleBulkStatus] MSISDNs:', msisdns);
    
    // Build items with old_status from sims array
    const items = msisdns.map(msisdn => {
      const sim = sims.find(s => s.MSISDN === msisdn);
      return {
        msisdn,
        cli: sim?.CLI || '',
        old_status: sim?.SIM_STATUS_CHANGE || ''
      };
    });
    
    updateStatusOptimistic(msisdns, status);
    showToast(`Запрос на изменение статуса ${msisdns.length} SIM...`, 'info');
    
    console.log('[handleBulkStatus] Calling ChangeStatus API with items:', items);
    const result = await ChangeStatus(items, status);
    console.log('[handleBulkStatus] API result:', result);
    
    if (result.success) {
      // If queued, we must have a requestId to poll; otherwise treat as immediate success.
      if (result.queued && result.requestId) {
        showToast(`✓ Запрос #${result.requestId} отправлен. Ожидание подтверждения...`, 'success');
        setSelectedSims(new Set());

        // Добавляем Job в очередь polling
        setPendingJobs(prev => [...prev, {
          requestId: result.requestId!,
          msisdns,
          targetStatus: status,
          startTime: Date.now(),
          attempts: 0
        }]);
      } else {
        showToast('✓ Статус обновлён', 'success');
        setSelectedSims(new Set());

        // Clear pending markers immediately
        setPendingStatuses(prev => {
          const newMap = new Map(prev);
          msisdns.forEach(m => newMap.delete(m));
          return newMap;
        });
        setSims(prev => prev.map(sim => {
          if (msisdns.includes(sim.MSISDN)) {
            return { ...sim, SIM_STATUS_CHANGE: status, _pending: false };
          }
          return sim;
        }));

        // Refresh to sync with backend data
        loadData(pagination.start, pagination.limit);
      }
    } else {
      showToast("Ошибка: " + (result.error || "Unknown error"), 'danger');
      loadData(pagination.start, pagination.limit);
      const newPending = new Map(pendingStatuses);
      msisdns.forEach(m => newPending.delete(m));
      setPendingStatuses(newPending);
    }
  };

  const handleSingleStatus = async (status: string) => {
    console.log('[handleSingleStatus] Called with status:', status, 'selectedSim:', selectedSim?.MSISDN);
    
    if (!selectedSim) {
      console.log('[handleSingleStatus] No SIM selected, returning');
      return;
    }
    
    if (!confirm(`Изменить статус ${selectedSim.MSISDN} на ${status}?`)) {
      console.log('[handleSingleStatus] User cancelled');
      return;
    }
    
    const msisdn = selectedSim.MSISDN;
    const oldStatus = selectedSim.SIM_STATUS_CHANGE || '';
    const cli = selectedSim.CLI || '';
    console.log('[handleSingleStatus] MSISDN:', msisdn, 'oldStatus:', oldStatus);
    
    updateStatusOptimistic([msisdn], status);
    setSelectedSim({ ...selectedSim, SIM_STATUS_CHANGE: status, _pending: true });
    showToast(`Запрос на изменение статуса ${msisdn}...`, 'info');
    
    const items = [{ msisdn, cli, old_status: oldStatus }];
    console.log('[handleSingleStatus] Calling ChangeStatus API with items:', items);
    const result = await ChangeStatus(items, status);
    console.log('[handleSingleStatus] API result:', result);
    
    if (result.success) {
      if (result.queued && result.requestId) {
        showToast(`✓ Запрос #${result.requestId} отправлен. Ожидание подтверждения...`, 'success');

        // Добавляем Job в очередь polling
        setPendingJobs(prev => [...prev, {
          requestId: result.requestId!,
          msisdns: [msisdn],
          targetStatus: status,
          startTime: Date.now(),
          attempts: 0
        }]);
      } else {
        showToast('✓ Статус обновлён', 'success');

        // Clear pending markers immediately
        setPendingStatuses(prev => {
          const newMap = new Map(prev);
          newMap.delete(msisdn);
          return newMap;
        });
        setSims(prev => prev.map(sim => {
          if (sim.MSISDN === msisdn) {
            return { ...sim, SIM_STATUS_CHANGE: status, _pending: false };
          }
          return sim;
        }));
        setSelectedSim((prev: any) => prev ? ({ ...prev, SIM_STATUS_CHANGE: status, _pending: false }) : prev);

        loadData(pagination.start, pagination.limit);
      }
    } else {
      showToast("Ошибка: " + (result.error || "Unknown error"), 'danger');
      loadData(pagination.start, pagination.limit);
      setPendingStatuses(prev => {
        const newMap = new Map(prev);
        newMap.delete(msisdn);
        return newMap;
      });
    }
  };

  const startEdit = () => {
    setEditValues({
      label1: selectedSim.CUSTOMER_LABEL_1 || '',
      label2: selectedSim.CUSTOMER_LABEL_2 || '',
      label3: selectedSim.CUSTOMER_LABEL_3 || ''
    });
    setEditMode(true);
  };

  const saveEdit = async () => {
    setLoading(true);
    const changes: string[] = [];
    const errors: string[] = [];
    
    try {
      // Label 1
      const oldLabel1 = selectedSim.CUSTOMER_LABEL_1 || '';
      if (editValues.label1 !== oldLabel1) {
        const result = await UpdateSim({
          msisdn: selectedSim.MSISDN,
          cli: selectedSim.CLI,
          field: "label_1",
          value: editValues.label1,
          old_value: oldLabel1
        });
        if (result.success) {
          const queuedText = result.queued ? ' (в очередь)' : '';
          changes.push(`Label 1: "${oldLabel1}" → "${editValues.label1}"${queuedText}`);
        } else {
          errors.push(`Label 1: ${result.error}`);
        }
      }
      
      // Label 2
      const oldLabel2 = selectedSim.CUSTOMER_LABEL_2 || '';
      if (editValues.label2 !== oldLabel2) {
        const result = await UpdateSim({
          msisdn: selectedSim.MSISDN,
          cli: selectedSim.CLI,
          field: "label_2",
          value: editValues.label2,
          old_value: oldLabel2
        });
        if (result.success) {
          const queuedText = result.queued ? ' (в очередь)' : '';
          changes.push(`Label 2: "${oldLabel2}" → "${editValues.label2}"${queuedText}`);
        } else {
          errors.push(`Label 2: ${result.error}`);
        }
      }
      
      // Label 3
      const oldLabel3 = selectedSim.CUSTOMER_LABEL_3 || '';
      if (editValues.label3 !== oldLabel3) {
        const result = await UpdateSim({
          msisdn: selectedSim.MSISDN,
          cli: selectedSim.CLI,
          field: "label_3",
          value: editValues.label3,
          old_value: oldLabel3
        });
        if (result.success) {
          const queuedText = result.queued ? ' (в очередь)' : '';
          changes.push(`Label 3: "${oldLabel3}" → "${editValues.label3}"${queuedText}`);
        } else {
          errors.push(`Label 3: ${result.error}`);
        }
      }
      
      if (errors.length > 0) {
        showToast(`Ошибки: ${errors.join('; ')}`, 'danger');
      } else if (changes.length > 0) {
        showToast(`Изменения сохранены:\n${changes.join('\n')}`, 'success');
      } else {
        showToast('Нет изменений для сохранения', 'info');
      }
      
      setEditMode(false);
      loadData(pagination.start, pagination.limit);
      setSelectedSim(null);
    } catch(e) {
      showToast("Ошибка обновления: " + e, 'danger');
    }
    setLoading(false);
  };

  const handleHeaderContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    const x = Math.min(e.clientX, window.innerWidth - 250);
    const y = Math.min(e.clientY, window.innerHeight - 400);
    setColumnMenuPos({ x, y });
    setShowColumnMenu(true);
  };

  const toggleColumn = (column: string) => {
    setVisibleColumns(prev => {
      if (prev.includes(column)) {
        if (prev.length === 1) {
          showToast('Должна быть хотя бы одна видимая колонка', 'warning');
          return prev;
        }
        return prev.filter(c => c !== column);
      }
      return [...prev, column];
    });
  };

  const resetColumns = () => {
    const defaultOrder = Object.keys(ALL_COLUMNS);
    const defaultVisible = Object.keys(ALL_COLUMNS).filter(k => ALL_COLUMNS[k].default);
    setColumnOrder(defaultOrder);
    setVisibleColumns(defaultVisible);
    setShowColumnMenu(false);
    showToast('Колонки сброшены к настройкам по умолчанию', 'info');
  };

  // Drag & Drop handlers для колонок
  const handleColumnDragStart = (e: React.DragEvent, column: string) => {
    setDraggedColumn(column);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleColumnDragOver = (e: React.DragEvent, targetColumn: string) => {
    e.preventDefault();
    if (!draggedColumn || draggedColumn === targetColumn) return;
    
    setColumnOrder(prev => {
      const newOrder = [...prev];
      const dragIdx = newOrder.indexOf(draggedColumn);
      const targetIdx = newOrder.indexOf(targetColumn);
      
      if (dragIdx === -1 || targetIdx === -1) return prev;
      
      newOrder.splice(dragIdx, 1);
      newOrder.splice(targetIdx, 0, draggedColumn);
      return newOrder;
    });
  };

  const handleColumnDragEnd = () => {
    setDraggedColumn(null);
  };

  // Вычисляем видимые колонки в правильном порядке
  const orderedVisibleColumns = columnOrder.filter(col => visibleColumns.includes(col));

  // Перемещение колонки вверх/вниз в меню
  const moveColumn = (column: string, direction: 'up' | 'down') => {
    setColumnOrder(prev => {
      const idx = prev.indexOf(column);
      if (idx === -1) return prev;
      
      const newIdx = direction === 'up' ? idx - 1 : idx + 1;
      if (newIdx < 0 || newIdx >= prev.length) return prev;
      
      const newOrder = [...prev];
      [newOrder[idx], newOrder[newIdx]] = [newOrder[newIdx], newOrder[idx]];
      return newOrder;
    });
  };

  const getCellValue = (sim: any, column: string) => {
    const config = ALL_COLUMNS[column];
    if (!config) return '-';
    
    const value = sim[config.field];
    
    if (column === 'SIM_STATUS_CHANGE') {
      return <StatusBadge status={value} isPending={sim._pending} syncStatus={sim.SYNC_STATUS} />;
    }
    
    return value || '-';
  };

  // Вычисляемые значения
  const currentListPage = Math.floor(pagination.start / pagination.limit) + 1;
  const totalPages = Math.ceil(total / pagination.limit);

  // ==================== РЕНДЕР ====================

  // Загрузка сессии
  if (isCheckingSession) {
    return (
      <div className="d-flex align-items-center justify-content-center vh-100 bg-dark">
        <div className="text-center">
          <div className="spinner-border text-primary mb-3" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted">Проверка сессии...</p>
        </div>
      </div>
    );
  }

  // Форма входа
  if (!isLoggedIn) {
    return (
      <div className="d-flex align-items-center justify-content-center vh-100 bg-dark">
        <div className="card p-4 shadow-lg" style={{width: '400px'}}>
          <div className="card-body">
            <h2 className="text-center mb-4 text-primary">EyesOn Login</h2>
            <form onSubmit={handleLogin}>
              <div className="mb-3">
                <label className="form-label text-light">Username</label>
                <input 
                  type="text" 
                  className="form-control" 
                  value={username} 
                  onChange={(e) => setUsername(e.target.value)} 
                  required 
                  autoFocus
                />
              </div>
              <div className="mb-3">
                <label className="form-label text-light">Password</label>
                <input 
                  type="password" 
                  className="form-control" 
                  value={password} 
                  onChange={(e) => setPassword(e.target.value)} 
                  required 
                />
              </div>
              <button type="submit" className="btn btn-primary w-100" disabled={loading}>
                {loading ? "Вход..." : "Войти"}
              </button>
            </form>
          </div>
        </div>
        
        <ToastContainer toasts={toasts} />
      </div>
    );
  }

  // Главный интерфейс
  return (
    <div className="container-fluid py-4">
      {/* Навигация */}
      <nav className="navbar navbar-expand navbar-dark bg-dark mb-4 rounded">
        <div className="container-fluid">
          <span className="navbar-brand">EyesOn</span>
          <ul className="navbar-nav">
            <li className="nav-item">
              <button 
                className={`nav-link btn btn-link ${navPage === 'sims' ? 'active fw-bold' : ''}`}
                onClick={() => setNavPage('sims')}
              >
                📱 SIM Cards
              </button>
            </li>
            <li className="nav-item">
              <button 
                className={`nav-link btn btn-link ${navPage === 'jobs' ? 'active fw-bold' : ''}`}
                onClick={() => setNavPage('jobs')}
              >
                📋 Jobs
              </button>
            </li>
            <li className="nav-item">
              <button 
                className={`nav-link btn btn-link ${navPage === 'queue' ? 'active fw-bold' : ''}`}
                onClick={() => setNavPage('queue')}
              >
                ⏳ Queue
              </button>
            </li>
            <li className="nav-item">
              <button 
                className={`nav-link btn btn-link ${navPage === 'stats' ? 'active fw-bold' : ''}`}
                onClick={() => setNavPage('stats')}
              >
                📊 Statistics
              </button>
            </li>
            <li className="nav-item">
              <button 
                className={`nav-link btn btn-link ${navPage === 'admin' ? 'active fw-bold' : ''}`}
                onClick={() => setNavPage('admin')}
              >
                ⚙️ Admin
              </button>
            </li>
            <li className="nav-item">
              <button 
                className={`nav-link btn btn-link ${navPage === 'profile' ? 'active fw-bold' : ''}`}
                onClick={() => setNavPage('profile')}
              >
                👤 Profile
              </button>
            </li>
          </ul>
          <div className="d-flex gap-2 align-items-center">
            <span className="badge bg-secondary">Total: {stats?.total || total}</span>
            <button 
              className="btn btn-outline-light btn-sm" 
              onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
              title={`Switch to ${theme === 'dark' ? 'Light' : 'Dark'} theme`}
            >
              {theme === 'dark' ? '☀️' : '🌙'}
            </button>
            <button className="btn btn-outline-light btn-sm" onClick={() => { 
              if (navPage === 'sims') { 
                loadData(pagination.start, pagination.limit);
                // Перезагружаем статистику через 2 сек
                setStatsLoaded(false);
                setTimeout(() => loadStats(true), 2000);
              }
              else { loadJobs(true); }  // force reload
            }}>
              {loading ? <span className="spinner-border spinner-border-sm"></span> : 'Refresh'}
            </button>
            <button className="btn btn-outline-danger btn-sm" onClick={handleLogout} title="Выйти">
              🚪
            </button>
          </div>
        </div>
      </nav>

      {/* Страница SIM */}
      {navPage === 'sims' && (
        <>
          {/* Заголовок */}
          <div className="d-flex justify-content-between align-items-center mb-4">
            <h1 className="h3 mb-0 text-white">SIM Management</h1>
          </div>

          {/* Статистика */}
          {stats && (
            <div className="row row-cols-2 row-cols-md-5 g-3 mb-4">
              <div className="col">
                <div className="card bg-dark border-primary h-100">
                  <div className="card-body">
                    <h6 className="card-subtitle mb-2 text-muted">Total SIMs</h6>
                    <h3 className="card-title text-primary">{stats.total || 0}</h3>
                  </div>
                </div>
              </div>
              <div className="col">
                <div className="card bg-dark border-secondary h-100">
                  <div className="card-body">
                    <h6 className="card-subtitle mb-2 text-muted">Activated</h6>
                    <h3 className="card-title text-success">{stats.by_status?.Activated || 0}</h3>
                  </div>
                </div>
              </div>
              <div className="col">
                <div className="card bg-dark border-secondary h-100">
                  <div className="card-body">
                    <h6 className="card-subtitle mb-2 text-muted">Suspended</h6>
                    <h3 className="card-title text-warning">{stats.by_status?.Suspended || 0}</h3>
                  </div>
                </div>
              </div>
              <div className="col">
                <div className="card bg-dark border-secondary h-100">
                  <div className="card-body">
                    <h6 className="card-subtitle mb-2 text-muted">Terminated</h6>
                    <h3 className="card-title text-danger">{stats.by_status?.Terminated || 0}</h3>
                  </div>
                </div>
              </div>
              <div className="col">
                <div className="card bg-dark border-secondary h-100">
                  <div className="card-body">
                    <h6 className="card-subtitle mb-2 text-muted">In Session</h6>
                    <h3 className="card-title text-info">{stats.active_sessions || 0}</h3>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Массовые действия */}
          {selectedSims.size > 0 && (
            <div className="card mb-3 border-primary bg-dark">
              <div className="card-body py-2 d-flex justify-content-between align-items-center">
                <span className="text-primary fw-bold">{selectedSims.size} Selected</span>
                <div className="btn-group btn-group-sm">
                  <button className="btn btn-outline-success" onClick={() => handleBulkStatus('Activated')}>Activate</button>
                  <button className="btn btn-outline-warning" onClick={() => handleBulkStatus('Suspended')}>Suspend</button>
                  <button className="btn btn-outline-danger" onClick={() => handleBulkStatus('Terminated')}>Terminate</button>
                </div>
              </div>
            </div>
          )}

          {/* Поиск и фильтры */}
          <div className="card mb-4">
            <div className="card-body">
              <form onSubmit={handleSearch} className="row g-3 align-items-center">
                <div className="col-md-6 col-lg-4 flex-grow-1">
                  <div className="input-group">
                    <span className="input-group-text bg-dark border-secondary text-light">🔍</span>
                    <input 
                      type="text" 
                      className="form-control" 
                      placeholder="Search by CLI, MSISDN, Label..." 
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                    />
                  </div>
                </div>
                <div className="col-auto">
                  <select 
                    className="form-select"
                    value={statusFilter}
                    onChange={(e) => {
                      setStatusFilter(e.target.value);
                      loadData(0, pagination.limit, search, sort.by, sort.direction, e.target.value);
                    }}
                  >
                    <option value="">All Statuses</option>
                    <option value="Activated">🟢 Activated</option>
                    <option value="Suspended">🟡 Suspended</option>
                    <option value="Terminated">🔴 Terminated</option>
                  </select>
                </div>
                <div className="col-auto">
                  <button type="submit" className="btn btn-primary" disabled={loading}>
                    {loading ? "Searching..." : "Search"}
                  </button>
                </div>
              </form>
            </div>
          </div>

          {/* Таблица */}
          <div className="card mb-4">
            <div className="table-responsive">
              <table className="table table-dark table-hover mb-0">
                <thead onContextMenu={handleHeaderContextMenu}>
                  <tr>
                    <th style={{width: '40px'}}>
                      <input type="checkbox" className="form-check-input" 
                        checked={sims.length > 0 && selectedSims.size === sims.length}
                        onChange={toggleSelectAll}
                      />
                    </th>
                    {orderedVisibleColumns.map(col => {
                      const config = ALL_COLUMNS[col];
                      if (!config) return null;
                      const isSortable = config.sortable && (col === 'CLI' || col === 'MSISDN');
                      return (
                        <th 
                          key={col}
                          draggable
                          onDragStart={(e) => handleColumnDragStart(e, col)}
                          onDragOver={(e) => handleColumnDragOver(e, col)}
                          onDragEnd={handleColumnDragEnd}
                          onClick={() => isSortable && handleSort(col)} 
                          style={{ 
                            cursor: isSortable ? 'pointer' : 'grab',
                            opacity: draggedColumn === col ? 0.5 : 1,
                            background: draggedColumn === col ? '#495057' : undefined
                          }}
                          title={isSortable ? 'Click to sort, drag to reorder' : 'Drag to reorder, right-click for settings'}
                        >
                          {config.name}
                          {sort.by === col && (
                            <span className="ms-1">{sort.direction === 'ASC' ? '▲' : '▼'}</span>
                          )}
                        </th>
                      );
                    })}
                    <th className="text-center">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {sims.length > 0 ? (
                    sims.map((sim: any) => (
                      <tr 
                        key={sim.MSISDN} 
                        onClick={() => setSelectedSim(sim)}
                        className={sim._pending ? 'table-warning' : ''}
                        style={{ cursor: 'pointer' }}
                      >
                        <td onClick={(e) => e.stopPropagation()}>
                          <input type="checkbox" className="form-check-input"
                            checked={selectedSims.has(sim.MSISDN)}
                            onChange={() => toggleSelect(sim.MSISDN)}
                          />
                        </td>
                        {orderedVisibleColumns.map(col => (
                          <td key={col}>{getCellValue(sim, col)}</td>
                        ))}
                        <td className="text-center" onClick={(e) => e.stopPropagation()}>
                          <button 
                            className="btn btn-sm btn-outline-info"
                            onClick={() => setSelectedSim(sim)}
                            title="View Details"
                          >
                            👁
                          </button>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={orderedVisibleColumns.length + 2} className="text-center py-4 text-muted">
                        {loading ? 'Loading...' : 'No records found'}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            
            {/* Пагинация */}
            <div className="card-footer d-flex justify-content-between align-items-center py-3">
              <div className="d-flex align-items-center gap-2">
                <span className="text-muted small">Show:</span>
                <select className="form-select form-select-sm" style={{width: '70px'}} value={pagination.limit} onChange={handleLimitChange}>
                  <option value="25">25</option>
                  <option value="50">50</option>
                  <option value="100">100</option>
                  <option value="500">500</option>
                  <option value="1000">All</option>
                </select>
              </div>
              
              <nav className="d-flex align-items-center">
                <ul className="pagination pagination-sm mb-0 me-2">
                   <li className={`page-item ${pagination.start === 0 ? 'disabled' : ''}`}>
                    <button className="page-link" onClick={() => handlePageChange(0)} title="First Page">«</button>
                  </li>
                  <li className={`page-item ${pagination.start === 0 ? 'disabled' : ''}`}>
                    <button className="page-link" onClick={() => handlePageChange(pagination.start - pagination.limit)} title="Previous Page">‹</button>
                  </li>
                </ul>
                
                <div className="input-group input-group-sm me-2" style={{width: 'auto'}}>
                  <span className="input-group-text bg-dark border-secondary text-light">Page</span>
                  <input 
                    type="number" 
                    className="form-control bg-dark border-secondary text-light text-center"
                    style={{width: '60px'}}
                    value={currentListPage}
                    min={1}
                    max={totalPages || 1}
                    onChange={(e) => {
                      const val = parseInt(e.target.value);
                      if (!isNaN(val)) {
                         // Allow typing, will correct on blur or verify bounds
                         const page = Math.max(1, Math.min(val, totalPages || 1));
                         handlePageChange((page - 1) * pagination.limit);
                      }
                    }}
                  />
                  <span className="input-group-text bg-dark border-secondary text-light">of {totalPages || 1}</span>
                </div>

                <ul className="pagination pagination-sm mb-0">
                  <li className={`page-item ${pagination.start + pagination.limit >= total ? 'disabled' : ''}`}>
                    <button className="page-link" onClick={() => handlePageChange(pagination.start + pagination.limit)} title="Next Page">›</button>
                  </li>
                  <li className={`page-item ${pagination.start + pagination.limit >= total ? 'disabled' : ''}`}>
                    <button className="page-link" onClick={() => handlePageChange(((totalPages || 1) - 1) * pagination.limit)} title="Last Page">»</button>
                  </li>
                </ul>
              </nav>
            </div>
          </div>

          {/* Меню колонок */}
          {showColumnMenu && (
            <div 
              ref={columnMenuRef}
              className="dropdown-menu show"
              style={{
                position: 'fixed',
                left: columnMenuPos.x,
                top: columnMenuPos.y,
                zIndex: 9999,
                maxHeight: '500px',
                overflowY: 'auto',
                minWidth: '280px'
              }}
            >
              <h6 className="dropdown-header">⚙️ Column Settings</h6>
              <div className="px-3 py-1 text-muted small">
                <i>Drag headers to reorder • Check to show/hide</i>
              </div>
              <div className="dropdown-divider"></div>
              <div className="px-2">
                {columnOrder.map((key, index) => {
                  const config = ALL_COLUMNS[key];
                  if (!config) return null;
                  return (
                    <div 
                      className="d-flex align-items-center py-1 px-1 rounded"
                      key={key}
                      style={{ 
                        background: visibleColumns.includes(key) ? 'rgba(13, 110, 253, 0.1)' : 'transparent'
                      }}
                    >
                      <input 
                        className="form-check-input me-2" 
                        type="checkbox"
                        id={`col_${key}`}
                        checked={visibleColumns.includes(key)}
                        onChange={() => toggleColumn(key)}
                      />
                      <label 
                        className="form-check-label flex-grow-1" 
                        htmlFor={`col_${key}`}
                        style={{ cursor: 'pointer' }}
                      >
                        {config.name}
                      </label>
                      <div className="btn-group btn-group-sm ms-2">
                        <button 
                          className="btn btn-outline-secondary btn-sm py-0 px-1"
                          onClick={() => moveColumn(key, 'up')}
                          disabled={index === 0}
                          title="Move Up"
                        >
                          ▲
                        </button>
                        <button 
                          className="btn btn-outline-secondary btn-sm py-0 px-1"
                          onClick={() => moveColumn(key, 'down')}
                          disabled={index === columnOrder.length - 1}
                          title="Move Down"
                        >
                          ▼
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
              <div className="dropdown-divider"></div>
              <div className="px-3 py-2">
                <button className="btn btn-sm btn-outline-secondary w-100" onClick={resetColumns}>
                  🔄 Reset to Default
                </button>
              </div>
            </div>
          )}

          {/* Модалка деталей SIM */}
          {selectedSim && (
            <SimDetailModal
              sim={selectedSim}
              editMode={editMode}
              editValues={editValues}
              loading={loading}
              onClose={() => { setSelectedSim(null); setEditMode(false); }}
              onEditStart={startEdit}
              onEditSave={saveEdit}
              onEditValueChange={(key, value) => setEditValues({...editValues, [key]: value})}
              onStatusChange={handleSingleStatus}
            />
          )}
        </>
      )}

      {/* Страница Jobs - Pelephone API Jobs */}
      {navPage === 'jobs' && (
        <div>
          <div className="d-flex justify-content-between align-items-center mb-4">
            <h1 className="h3 mb-0 text-white">Pelephone Provisioning Jobs</h1>
            <button className="btn btn-outline-light btn-sm" onClick={() => { setJobsLoaded(false); loadJobs(true); }}>
              🔄 Refresh
            </button>
          </div>

          {/* Фильтры */}
          <div className="card mb-4">
            <div className="card-body">
              <div className="row g-3 align-items-end">
                <div className="col-md-3">
                  <label className="form-label text-muted">Job ID</label>
                  <input 
                    type="number" 
                    className="form-control" 
                    placeholder="Enter Job ID"
                    value={jobFilters.jobId}
                    onChange={(e) => setJobFilters({...jobFilters, jobId: e.target.value})}
                  />
                </div>
                <div className="col-md-3">
                  <label className="form-label text-muted">Status</label>
                  <select 
                    className="form-select"
                    value={jobFilters.status}
                    onChange={(e) => setJobFilters({...jobFilters, status: e.target.value})}
                  >
                    <option value="">All Statuses</option>
                    <option value="PENDING">Pending</option>
                    <option value="IN_PROGRESS">In Progress</option>
                    <option value="SUCCESS">Completed</option>
                    <option value="FAILED">Failed</option>
                  </select>
                </div>
                <div className="col-md-3">
                  <button className="btn btn-primary" onClick={() => { setJobsLoaded(false); loadJobs(true); }}>
                    🔍 Apply Filters
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* Таблица Jobs */}
          <div className="card">
            <div className="table-responsive">
              <table className="table table-dark table-hover mb-0">
                <thead>
                  <tr>
                    <th>Job ID</th>
                    <th>Action Type</th>
                    <th>Change</th>
                    <th>Status</th>
                    <th>Created</th>
                    <th>Last Updated</th>
                    <th>Details</th>
                  </tr>
                </thead>
                <tbody>
                  {jobs.length > 0 ? (
                    jobs.map((job: any) => {
                      const actions = job.actions || [];
                      const firstAction = actions.length > 0 ? actions[0] : {};
                      const actionType = firstAction.requestType || job.actionType || '-';
                      const initialValue = firstAction.initialValue || '';
                      const targetValue = firstAction.targetValue || job.targetValue || '-';
                      const status = firstAction.status || job.jobStatus || job.status || 'PENDING';
                      const message = firstAction.errorDesc || '';
                      const changeDisplay = initialValue ? `${initialValue} → ${targetValue}` : targetValue;

                      return (
                        <tr key={job.jobId}>
                          <td><strong>{job.jobId}</strong></td>
                          <td><small>{actionType.replace(/_/g, ' ')}</small></td>
                          <td><code className="text-info">{changeDisplay}</code></td>
                          <td><JobStatusBadge status={status} /></td>
                          <td><small>{formatDate(job.requestTime)}</small></td>
                          <td><small>{formatDate(job.lastActionTime)}</small></td>
                          <td>
                            <button 
                              className="btn btn-outline-info btn-sm py-0 px-1"
                              onClick={() => {
                                const details = JSON.stringify(job, null, 2);
                                alert(`Job #${job.jobId} Details:\n\n${details}`);
                              }}
                              title="View full job details"
                            >
                              👁 {actions.length > 1 ? `(${actions.length})` : 'View'}
                            </button>
                            {message && message !== 'Success' && (
                              <small className="ms-2 text-warning">{message}</small>
                            )}
                          </td>
                        </tr>
                      );
                    })
                  ) : (
                    <tr>
                      <td colSpan={7} className="text-center py-4 text-muted">
                        {loading ? 'Loading...' : 'No jobs found from Pelephone API'}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            {/* Пагинация */}
            {allJobs.length > 0 && (
              <div className="card-footer d-flex justify-content-between align-items-center py-3">
                <div className="d-flex align-items-center gap-3">
                  <span className="text-muted small">
                    Records {(jobsPage - 1) * jobsPerPage + 1}-{Math.min(jobsPage * jobsPerPage, allJobs.length)} of <strong>{allJobs.length}</strong>
                  </span>
                  <select 
                    className="form-select form-select-sm" 
                    style={{width: 'auto'}}
                    value={jobsPerPage}
                    onChange={(e) => { setJobsPerPage(parseInt(e.target.value)); setJobsPage(1); }}
                  >
                    <option value="10">10 / page</option>
                    <option value="25">25 / page</option>
                    <option value="50">50 / page</option>
                    <option value="100">100 / page</option>
                  </select>
                </div>
                <nav>
                  <ul className="pagination pagination-sm mb-0">
                    <li className={`page-item ${jobsPage <= 1 ? 'disabled' : ''}`}>
                      <button className="page-link" onClick={() => setJobsPage(1)} disabled={jobsPage <= 1}>«</button>
                    </li>
                    <li className={`page-item ${jobsPage <= 1 ? 'disabled' : ''}`}>
                      <button className="page-link" onClick={() => setJobsPage(p => p - 1)} disabled={jobsPage <= 1}>‹</button>
                    </li>
                    {Array.from({ length: Math.min(5, jobsTotalPages) }, (_, i) => {
                      let pageNum;
                      if (jobsTotalPages <= 5) {
                        pageNum = i + 1;
                      } else if (jobsPage <= 3) {
                        pageNum = i + 1;
                      } else if (jobsPage >= jobsTotalPages - 2) {
                        pageNum = jobsTotalPages - 4 + i;
                      } else {
                        pageNum = jobsPage - 2 + i;
                      }
                      return (
                        <li key={pageNum} className={`page-item ${jobsPage === pageNum ? 'active' : ''}`}>
                          <button className="page-link" onClick={() => setJobsPage(pageNum)}>{pageNum}</button>
                        </li>
                      );
                    })}
                    <li className={`page-item ${jobsPage >= jobsTotalPages ? 'disabled' : ''}`}>
                      <button className="page-link" onClick={() => setJobsPage(p => p + 1)} disabled={jobsPage >= jobsTotalPages}>›</button>
                    </li>
                    <li className={`page-item ${jobsPage >= jobsTotalPages ? 'disabled' : ''}`}>
                      <button className="page-link" onClick={() => setJobsPage(jobsTotalPages)} disabled={jobsPage >= jobsTotalPages}>»</button>
                    </li>
                  </ul>
                </nav>
              </div>
            )}
          </div>
        </div>
      )}

      {navPage === 'queue' && <QueueView />}

      {/* Страница Statistics */}
      {navPage === 'stats' && (
        <div>
          <div className="d-flex justify-content-between align-items-center mb-4">
            <h1 className="h3 mb-0 text-white">📊 Statistics Dashboard</h1>
            <button className="btn btn-outline-light btn-sm" onClick={() => { setStatsLoaded(false); loadStats(true); }}>
              🔄 Refresh Stats
            </button>
          </div>


          {/* Summary Cards */}
          <div className="row g-4 mb-4">
            <div className="col-md-3">
              <div className="card bg-primary text-white h-100">
                <div className="card-body text-center">
                  <h6 className="card-subtitle mb-2 opacity-75">Total SIM Cards</h6>
                  <h2 className="card-title display-4">{stats?.total || 0}</h2>
                </div>
              </div>
            </div>
            <div className="col-md-3">
              <div className="card bg-success text-white h-100">
                <div className="card-body text-center">
                  <h6 className="card-subtitle mb-2 opacity-75">Activated</h6>
                  <h2 className="card-title display-4">{stats?.by_status?.Activated || 0}</h2>
                </div>
              </div>
            </div>
            <div className="col-md-3">
              <div className="card bg-warning text-dark h-100">
                <div className="card-body text-center">
                  <h6 className="card-subtitle mb-2 opacity-75">Suspended</h6>
                  <h2 className="card-title display-4">{stats?.by_status?.Suspended || 0}</h2>
                </div>
              </div>
            </div>
            <div className="col-md-3">
              <div className="card bg-danger text-white h-100">
                <div className="card-body text-center">
                  <h6 className="card-subtitle mb-2 opacity-75">Terminated</h6>
                  <h2 className="card-title display-4">{stats?.by_status?.Terminated || 0}</h2>
                </div>
              </div>
            </div>
          </div>

          {/* Status Distribution & Usage Stats */}
          <div className="row g-4 mb-4">
            <div className="col-md-6">
              <div className="card bg-dark border-secondary h-100">
                <div className="card-header border-secondary">
                  <h5 className="mb-0">📈 Status Distribution</h5>
                </div>
                <div className="card-body">
                  {stats?.by_status && Object.entries(stats.by_status).map(([status, count]: [string, any]) => {
                    const percentage = stats.total > 0 ? ((count / stats.total) * 100).toFixed(1) : '0';
                    const color = status === 'Activated' ? 'success' : status === 'Suspended' ? 'warning' : status === 'Terminated' ? 'danger' : 'secondary';
                    return (
                      <div key={status} className="mb-3">
                        <div className="d-flex justify-content-between mb-1">
                          <span>{status}</span>
                          <span className="text-muted">{count} ({percentage}%)</span>
                        </div>
                        <div className="progress" style={{height: '8px'}}>
                          <div className={`progress-bar bg-${color}`} style={{width: `${percentage}%`}}></div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            <div className="col-md-6">
              <div className="card bg-dark border-secondary h-100">
                <div className="card-header border-secondary">
                  <h5 className="mb-0">🌐 Top Rate Plans</h5>
                </div>
                <div className="card-body">
                  {stats?.by_rate_plan && Object.entries(stats.by_rate_plan)
                    .sort((a: any, b: any) => b[1] - a[1])
                    .slice(0, 5)
                    .map(([plan, count]: [string, any]) => (
                      <div key={plan} className="d-flex justify-content-between py-2 border-bottom border-secondary">
                        <span className="text-truncate" style={{maxWidth: '70%'}}>{plan || 'No Plan'}</span>
                        <span className="badge bg-info">{count}</span>
                      </div>
                    ))
                  }
                </div>
              </div>
            </div>
          </div>

          {/* Jobs Overview */}
          <div className="row g-4">
            <div className="col-12">
              <div className="card bg-dark border-secondary">
                <div className="card-header border-secondary d-flex justify-content-between align-items-center">
                  <h5 className="mb-0">📋 Recent Jobs Activity</h5>
                  <button className="btn btn-outline-info btn-sm" onClick={() => setNavPage('jobs')}>
                    View All Jobs →
                  </button>
                </div>
                <div className="card-body">
                  <div className="row text-center">
                    <div className="col-md-3">
                      <h4 className="text-info">{allJobs.length}</h4>
                      <small className="text-muted">Total Jobs</small>
                    </div>
                    <div className="col-md-3">
                      <h4 className="text-warning">{allJobs.filter((j: any) => j.jobStatus === 'PENDING' || (j.actions && j.actions[0]?.status === 'PENDING')).length}</h4>
                      <small className="text-muted">Pending</small>
                    </div>
                    <div className="col-md-3">
                      <h4 className="text-success">{allJobs.filter((j: any) => j.jobStatus === 'SUCCESS' || (j.actions && j.actions[0]?.status === 'SUCCESS')).length}</h4>
                      <small className="text-muted">Completed</small>
                    </div>
                    <div className="col-md-3">
                      <h4 className="text-danger">{allJobs.filter((j: any) => j.jobStatus === 'FAILED' || (j.actions && j.actions[0]?.status === 'FAILED')).length}</h4>
                      <small className="text-muted">Failed</small>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Страница Admin */}
      {navPage === 'admin' && (
        <div>
          <div className="d-flex justify-content-between align-items-center mb-4">
            <h1 className="h3 mb-0 text-white">⚙️ Admin Panel</h1>
            <button className="btn btn-outline-light btn-sm" onClick={() => { setUsersLoaded(false); loadUsers(); }}>
              🔄 Refresh
            </button>
          </div>

          {/* User Management Section */}
          <div className="card bg-dark border-secondary mb-4">
            <div className="card-header border-secondary d-flex justify-content-between align-items-center">
              <h5 className="mb-0">👥 User Management</h5>
              <button className="btn btn-success btn-sm" onClick={openCreateUserModal}>
                ➕ Add User
              </button>
            </div>
            <div className="card-body p-0">
              <div className="table-responsive">
                <table className="table table-dark table-hover mb-0">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Username</th>
                      <th>Email</th>
                      <th>Role</th>
                      <th>Status</th>
                      <th>Created</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {users.length > 0 ? users.map(user => (
                      <tr key={user.id}>
                        <td>{user.id}</td>
                        <td><strong>{user.username}</strong></td>
                        <td>{user.email}</td>
                        <td>
                          <span className={`badge ${user.role === 'Administrator' ? 'bg-danger' : user.role === 'Moderator' ? 'bg-warning' : 'bg-secondary'}`}>
                            {user.role}
                          </span>
                        </td>
                        <td>
                          <span className={`badge ${user.is_active ? 'bg-success' : 'bg-secondary'}`}>
                            {user.is_active ? 'Active' : 'Inactive'}
                          </span>
                        </td>
                        <td><small>{new Date(user.created_at).toLocaleDateString()}</small></td>
                        <td>
                          <div className="btn-group btn-group-sm">
                            <button className="btn btn-outline-primary py-0" onClick={() => openEditUserModal(user)} title="Edit">
                              ✏️
                            </button>
                            <button className="btn btn-outline-warning py-0" onClick={() => openResetPasswordModal(user.id)} title="Reset Password">
                              🔑
                            </button>
                            <button 
                              className={`btn ${user.is_active ? 'btn-outline-secondary' : 'btn-outline-success'} py-0`}
                              onClick={() => handleToggleUserActive(user)}
                              title={user.is_active ? 'Deactivate' : 'Activate'}
                            >
                              {user.is_active ? '🚫' : '✅'}
                            </button>
                            <button className="btn btn-outline-danger py-0" onClick={() => handleDeleteUser(user)} title="Delete">
                              🗑️
                            </button>
                          </div>
                        </td>
                      </tr>
                    )) : (
                      <tr>
                        <td colSpan={7} className="text-center py-4 text-muted">
                          {loading ? 'Loading...' : 'No users found'}
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
            <div className="card-footer border-secondary">
              <small className="text-muted">Total users: {users.length} | Active: {users.filter(u => u.is_active).length}</small>
            </div>
          </div>

          <div className="row g-4">
            {/* System Settings Card */}
            <div className="col-md-6">
              <div className="card bg-dark border-secondary h-100">
                <div className="card-header border-secondary">
                  <h5 className="mb-0">🔧 System Settings</h5>
                </div>
                <div className="card-body">
                  <ul className="list-unstyled">
                    <li className="py-2 border-bottom border-secondary">
                      <strong>API Endpoint:</strong> <code className="text-info">{window.location.origin}</code>
                    </li>
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Session Timeout:</strong> <span className="text-muted">24 hours</span>
                    </li>
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Documentation:</strong>
                      <div className="mt-1 d-flex flex-wrap gap-3">
                        <a href="/swagger.html" target="_blank" className="text-info text-decoration-none" title="Local API Documentation">
                          📄 Swagger
                        </a>
                        <a href="https://eot-portal.pelephone.co.il:8888/ipa/apis/docs" target="_blank" className="text-warning text-decoration-none" title="Pelephone EyesOnT API">
                          🌐 Pelephone API
                        </a>
                        <a href="https://github.com/alexgavs/eyeson-go/blob/main/docs/ARCHITECTURE.md" target="_blank" className="text-info text-decoration-none" title="Architecture Documentation">
                          🏗️ Architecture
                        </a>
                        <a href="https://github.com/alexgavs/eyeson-go/blob/main/docs/DEVELOPMENT_RULES.md" target="_blank" className="text-info text-decoration-none" title="Development Rules">
                          📋 Dev Rules
                        </a>
                      </div>
                    </li>
                    <li className="py-2">
                      <strong>Available Roles:</strong> {roles.map(r => (
                        <span key={r.id} className="badge bg-secondary me-1">{r.name}</span>
                      ))}
                    </li>
                  </ul>
                </div>
              </div>
            </div>

            {/* API Status Card */}
            <div className="col-md-6">
              <div className="card bg-dark border-secondary h-100">
                <div className="card-header border-secondary d-flex justify-content-between align-items-center">
                  <h5 className="mb-0">🌐 API Status</h5>
                  {isAdmin && (
                    <button 
                      className="btn btn-outline-light btn-sm" 
                      onClick={loadAPIStatus}
                      disabled={apiStatusLoading}
                    >
                      {apiStatusLoading ? '⏳' : '🔄'} Refresh
                    </button>
                  )}
                </div>
                <div className="card-body">
                  <ul className="list-unstyled">
                    <li className="py-2 border-bottom border-secondary d-flex justify-content-between">
                      <span>EyesOn API (Pelephone)</span>
                      <span className={`badge ${apiStatus?.eyeson_api?.status === 'online' ? 'bg-success' : apiStatus ? 'bg-danger' : 'bg-secondary'}`}>
                        ● {apiStatus?.eyeson_api?.status || 'Unknown'}
                        {apiStatus?.eyeson_api?.response_time_ms ? ` (${apiStatus.eyeson_api.response_time_ms}ms)` : ''}
                      </span>
                    </li>
                    <li className="py-2 border-bottom border-secondary d-flex justify-content-between">
                      <span>Go Backend</span>
                      <span className={`badge ${apiStatus?.go_backend?.status === 'online' ? 'bg-success' : apiStatus ? 'bg-danger' : 'bg-secondary'}`}>
                        ● {apiStatus?.go_backend?.status || 'Unknown'}
                      </span>
                    </li>
                    <li className="py-2 border-bottom border-secondary d-flex justify-content-between">
                      <span>Database</span>
                      <span className={`badge ${apiStatus?.database?.status === 'online' ? 'bg-success' : apiStatus ? 'bg-danger' : 'bg-secondary'}`}>
                        ● {apiStatus?.database?.status || 'Unknown'}
                      </span>
                    </li>
                  </ul>
                  
                  {/* Admin only: API Details - show even on error */}
                  {isAdmin && apiStatus?.eyeson_api?.details && (
                    <div className="mt-3 p-3 bg-black rounded">
                      <h6 className="text-warning mb-2">🔐 API Connection Details (Admin)</h6>
                      <table className="table table-sm table-dark mb-0">
                        <tbody>
                          <tr>
                            <td className="text-muted">API URL:</td>
                            <td><code className="text-info">{apiStatus.eyeson_api.details.api_url}</code></td>
                          </tr>
                          <tr>
                            <td className="text-muted">API User:</td>
                            <td><code className="text-success">{apiStatus.eyeson_api.details.api_user}</code></td>
                          </tr>
                          {apiStatus.eyeson_api.details.total_sims && (
                            <tr>
                              <td className="text-muted">Total SIMs:</td>
                              <td><code className="text-light">{apiStatus.eyeson_api.details.total_sims}</code></td>
                            </tr>
                          )}
                          {apiStatus.eyeson_api.details.api_result && (
                            <tr>
                              <td className="text-muted">API Result:</td>
                              <td><code className={apiStatus.eyeson_api.details.api_result === 'SUCCESS' ? 'text-success' : 'text-danger'}>{apiStatus.eyeson_api.details.api_result}</code></td>
                            </tr>
                          )}
                          {apiStatus.eyeson_api.details.error_type && (
                            <tr>
                              <td className="text-muted">Error Type:</td>
                              <td><code className="text-danger">{apiStatus.eyeson_api.details.error_type}</code></td>
                            </tr>
                          )}
                          {apiStatus.eyeson_api.details.hint && (
                            <tr>
                              <td className="text-muted">Hint:</td>
                              <td><small className="text-warning">{apiStatus.eyeson_api.details.hint}</small></td>
                            </tr>
                          )}
                          <tr>
                            <td className="text-muted">Last Check:</td>
                            <td><code className="text-light">{new Date(apiStatus.last_checked).toLocaleString()}</code></td>
                          </tr>
                        </tbody>
                      </table>

                      <div className="mt-3 p-3 bg-dark rounded border border-secondary">
                        <div className="d-flex align-items-center justify-content-between">
                          <h6 className="text-info mb-0">🌐 Upstream Provider</h6>
                          <button className="btn btn-outline-light btn-sm" onClick={loadUpstream} disabled={upstreamLoading}>
                            {upstreamLoading ? '⏳' : '🔄'}
                          </button>
                        </div>

                        <div className="mt-2">
                          <label className="form-label text-muted small mb-1">Select provider (applies after restart)</label>
                          <select
                            className="form-select form-select-sm"
                            value={upstreamSelectedDraft}
                            onChange={(e) => setUpstreamSelectedDraft(e.target.value as any)}
                            disabled={upstreamSaving || upstreamLoading}
                          >
                            <option value="pelephone">Pelephone server (env)</option>
                            <option value="simulator">Simulator server (env)</option>
                          </select>

                          {upstreamCfg?.options && (
                            <div className="mt-2 small">
                              <div className="text-muted">Pelephone URL: <code className="text-info">{upstreamCfg.options.pelephone?.base_url}</code></div>
                              <div className="text-muted">Simulator URL: <code className="text-info">{upstreamCfg.options.simulator?.base_url}</code></div>
                            </div>
                          )}

                          <div className="d-grid gap-2 mt-2">
                            <button
                              className="btn btn-outline-primary btn-sm"
                              disabled={
                                upstreamSaving || upstreamLoading ||
                                !upstreamCfg ||
                                upstreamSelectedDraft === upstreamCfg.selected
                              }
                              onClick={async () => {
                                setUpstreamSaving(true);
                                try {
                                  const saved = await SetUpstream(upstreamSelectedDraft);
                                  setUpstreamCfg(saved);
                                  setUpstreamRestartRequired(true);
                                  showToast('Upstream selection saved. Restart server to apply.', 'warning');
                                } catch (e: any) {
                                  showToast(e.message || 'Failed to save upstream selection', 'danger');
                                } finally {
                                  setUpstreamSaving(false);
                                }
                              }}
                            >
                              {upstreamSaving ? 'Saving...' : 'Apply (Restart Required)'}
                            </button>
                          </div>

                          {upstreamRestartRequired && (
                            <div className="alert alert-warning mt-2 mb-0 py-2">
                              <small><strong>Restart required:</strong> перезапусти Go сервер, чтобы применить выбор.</small>
                            </div>
                          )}
                        </div>
                      </div>

            <div className="mt-3 p-3 bg-dark rounded border border-secondary">
            <div className="d-flex align-items-center justify-content-between">
              <h6 className="text-info mb-0">🗄️ Sync Databases</h6>
              <button className="btn btn-outline-light btn-sm" onClick={loadManualSyncStatus} disabled={manualSyncLoading}>
              {manualSyncLoading ? '⏳' : '🔄'}
              </button>
            </div>

            <div className="mt-2 small text-muted">
              Источник истины: <code className="text-info">Pelephone</code>. Можно синхронизировать в локальную БД и (опционально) залить данные в симулятор.
            </div>

            {manualSyncStatus && (
              <div className="mt-2 small">
              {(() => {
                const isIdle = !manualSyncStatus.running && !manualSyncStatus.started_at && !manualSyncStatus.finished_at;
                const cls = isIdle ? 'text-light' : (manualSyncStatus.running ? 'text-warning' : (manualSyncStatus.last_success ? 'text-success' : 'text-danger'));
                const txt = isIdle ? 'IDLE' : (manualSyncStatus.running ? 'RUNNING' : (manualSyncStatus.last_success ? 'OK' : 'ERROR'));
                return (
                <div className="text-muted">Status: <code className={cls}>{txt}</code></div>
                );
              })()}
              {manualSyncStatus.started_at && (
                <div className="text-muted">Started: <code className="text-light">{new Date(manualSyncStatus.started_at).toLocaleString()}</code></div>
              )}
              {manualSyncStatus.finished_at && (
                <div className="text-muted">Finished: <code className="text-light">{new Date(manualSyncStatus.finished_at).toLocaleString()}</code></div>
              )}
              <div className="text-muted">Processed: <code className="text-light">{manualSyncStatus.last_processed}</code> | Duration: <code className="text-light">{manualSyncStatus.last_duration_ms}ms</code></div>

              {manualSyncStatus.clear_local_db && (
                <div className="text-muted">
                  Cleared before sync: <code className="text-warning">{manualSyncStatus.deleted_before_sync ?? 0}</code> records deleted
                </div>
              )}

              {manualSyncStatus.simulator_requested && (
                <div className="text-muted">
                  Simulator push:{' '}
                  <code className={manualSyncStatus.simulator_last_push_ok ? 'text-success' : (manualSyncStatus.simulator_last_error ? 'text-danger' : 'text-light')}>
                    {manualSyncStatus.simulator_last_push_ok ? 'OK' : (manualSyncStatus.simulator_last_error ? 'ERROR' : 'PENDING')}
                  </code>
                  {' '}| Pushed: <code className="text-light">{manualSyncStatus.simulator_last_pushed ?? 0}</code>
                  {' '}| Duration: <code className="text-light">{manualSyncStatus.simulator_duration_ms ?? 0}ms</code>
                  {manualSyncStatus.simulator_base_url && (
                    <> | URL: <code className="text-info">{manualSyncStatus.simulator_base_url}</code></>
                  )}
                </div>
              )}

              {manualSyncStatus.last_error && (
                <div className="text-muted">Error: <code className="text-danger">{manualSyncStatus.last_error}</code></div>
              )}
              {!manualSyncStatus.last_error && manualSyncStatus.simulator_last_error && (
                <div className="text-muted">Simulator error: <code className="text-danger">{manualSyncStatus.simulator_last_error}</code></div>
              )}
              </div>
            )}

            <div className="d-grid gap-2 mt-2">
              <button
              className="btn btn-outline-warning btn-sm"
              disabled={manualSyncStarting || manualSyncStatus?.running}
              onClick={async () => {
                if (!window.confirm('Запустить полную синхронизацию из Pelephone в локальную БД?')) return;
                setManualSyncStarting(true);
                try {
                await TriggerManualFullSync({ pushSimulator: false });
                showToast('Sync started (Pelephone → local DB).', 'info');
                loadManualSyncStatus();
                } catch (e: any) {
                showToast(e.message || 'Failed to start sync', 'danger');
                } finally {
                setManualSyncStarting(false);
                }
              }}
              >
              {manualSyncStarting ? 'Starting...' : (manualSyncStatus?.running ? 'Sync Running...' : 'Sync Now (Pelephone → Local)')}
              </button>

              <button
                className="btn btn-outline-danger btn-sm"
                disabled={manualSyncStarting || manualSyncStatus?.running}
                onClick={async () => {
                  if (!window.confirm('⚠️ ВНИМАНИЕ: Это удалит ВСЕ записи из локальной БД и загрузит заново с Pelephone!\n\nПродолжить?')) return;
                  setManualSyncStarting(true);
                  try {
                    await TriggerManualFullSync({ clearLocalDB: true, pushSimulator: false });
                    showToast('Clear & Sync started (DELETE ALL → Pelephone → local DB).', 'warning');
                    loadManualSyncStatus();
                  } catch (e: any) {
                    showToast(e.message || 'Failed to start sync', 'danger');
                  } finally {
                    setManualSyncStarting(false);
                  }
                }}
              >
                {manualSyncStarting ? 'Starting...' : (manualSyncStatus?.running ? 'Sync Running...' : '🗑️ Clear & Sync (Delete Local → Reload)')}
              </button>

              <button
                className="btn btn-outline-info btn-sm"
                disabled={manualSyncStarting || manualSyncStatus?.running}
                onClick={async () => {
                  if (!window.confirm('Синхронизировать из Pelephone в локальную БД и залить в симулятор? (требуется запущенный симулятор)')) return;
                  setManualSyncStarting(true);
                  try {
                    await TriggerManualFullSync({ pushSimulator: true });
                    showToast('Sync started (Pelephone → local DB → simulator).', 'info');
                    loadManualSyncStatus();
                  } catch (e: any) {
                    showToast(e.message || 'Failed to start sync', 'danger');
                  } finally {
                    setManualSyncStarting(false);
                  }
                }}
              >
                {manualSyncStarting ? 'Starting...' : (manualSyncStatus?.running ? 'Sync Running...' : 'Sync + Push to Simulator')}
              </button>
            </div>
            </div>
                    </div>
                  )}
                  
                  {apiStatus?.eyeson_api?.error && (
                    <div className="alert alert-danger mt-2 mb-0 py-2">
                      <small><strong>Error:</strong> {apiStatus.eyeson_api.error}</small>
                    </div>
                  )}
                  
                  <div className="d-grid gap-2 mt-3">
                    <button 
                      className="btn btn-outline-success btn-sm" 
                      onClick={() => { loadData(pagination.start, pagination.limit); if (isAdmin) loadAPIStatus(); }}
                    >
                      🔗 Test Connection
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* User Modal */}
      {showUserModal && (
        <div className="modal show d-block" style={{backgroundColor: 'rgba(0,0,0,0.5)'}}>
          <div 
            className="modal-dialog" 
            style={{
              transform: `translate(${modalPosition.x}px, ${modalPosition.y}px)`,
              transition: isDragging ? 'none' : 'transform 0.1s'
            }}
          >
            <div className="modal-content bg-dark text-white">
              <div 
                className="modal-header border-secondary" 
                style={{ cursor: 'move', userSelect: 'none' }}
                onMouseDown={handleMouseDown}
              >
                <h5 className="modal-title">{editingUser ? '✏️ Edit User' : '➕ Create User'}</h5>
                <button type="button" className="btn-close btn-close-white" onClick={() => setShowUserModal(false)}></button>
              </div>
              <div className="modal-body">
                <div className="mb-3">
                  <label className="form-label">Username *</label>
                  <input 
                    type="text" 
                    className="form-control bg-dark text-white border-secondary" 
                    value={userForm.username}
                    onChange={e => setUserForm({...userForm, username: e.target.value})}
                  />
                </div>
                <div className="mb-3">
                  <label className="form-label">Email *</label>
                  <input 
                    type="email" 
                    className="form-control bg-dark text-white border-secondary" 
                    value={userForm.email}
                    onChange={e => setUserForm({...userForm, email: e.target.value})}
                  />
                </div>
                {!editingUser && (
                  <div className="mb-3">
                    <label className="form-label">Password *</label>
                    <input 
                      type="password" 
                      className="form-control bg-dark text-white border-secondary" 
                      value={userForm.password}
                      onChange={e => setUserForm({...userForm, password: e.target.value})}
                      placeholder="Min 6 characters"
                    />
                  </div>
                )}
                <div className="mb-3">
                  <label className="form-label">Role</label>
                  <select 
                    className="form-select bg-dark text-white border-secondary"
                    value={userForm.role}
                    onChange={e => setUserForm({...userForm, role: e.target.value})}
                  >
                    {roles.length > 0 ? roles.map(role => (
                      <option key={role.id} value={role.name}>{role.name}</option>
                    )) : (
                      <>
                        <option value="Administrator">Administrator</option>
                        <option value="Moderator">Moderator</option>
                        <option value="Viewer">Viewer</option>
                      </>
                    )}
                  </select>
                </div>
              </div>
              <div className="modal-footer border-secondary">
                <button type="button" className="btn btn-secondary" onClick={() => setShowUserModal(false)}>Cancel</button>
                <button type="button" className="btn btn-primary" onClick={handleSaveUser} disabled={loading}>
                  {loading ? 'Saving...' : (editingUser ? 'Update' : 'Create')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Reset Password Modal */}
      {showResetPasswordModal && (
        <div className="modal show d-block" style={{backgroundColor: 'rgba(0,0,0,0.5)'}}>
          <div className="modal-dialog modal-sm">
            <div className="modal-content bg-dark text-white">
              <div className="modal-header border-secondary">
                <h5 className="modal-title">🔑 Reset Password</h5>
                <button type="button" className="btn-close btn-close-white" onClick={() => setShowResetPasswordModal(false)}></button>
              </div>
              <div className="modal-body">
                <div className="mb-3">
                  <label className="form-label">New Password</label>
                  <input 
                    type="password" 
                    className="form-control bg-dark text-white border-secondary" 
                    value={newPassword}
                    onChange={e => setNewPassword(e.target.value)}
                    placeholder="Min 6 characters"
                  />
                </div>
              </div>
              <div className="modal-footer border-secondary">
                <button type="button" className="btn btn-secondary" onClick={() => setShowResetPasswordModal(false)}>Cancel</button>
                <button type="button" className="btn btn-warning" onClick={handleResetPassword} disabled={loading}>
                  {loading ? 'Resetting...' : 'Reset Password'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Страница Profile */}
      {navPage === 'profile' && (
        <div>
          <div className="d-flex justify-content-between align-items-center mb-4">
            <h1 className="h3 mb-0 text-white">👤 User Profile</h1>
          </div>

          <div className="row g-4">
            {/* Profile Info Card */}
            <div className="col-md-6">
              <div className="card bg-dark border-secondary">
                <div className="card-header border-secondary">
                  <h5 className="mb-0">📋 Profile Information</h5>
                </div>
                <div className="card-body">
                  <div className="text-center mb-4">
                    <div className="rounded-circle bg-primary d-inline-flex align-items-center justify-content-center" style={{width: '80px', height: '80px', fontSize: '2rem'}}>
                      👤
                    </div>
                  </div>
                  <ul className="list-unstyled">
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Username:</strong> <span className="text-info">{localStorage.getItem('username') || 'admin'}</span>
                    </li>
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Role:</strong> <span className="badge bg-primary">Administrator</span>
                    </li>
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Email:</strong> <span className="text-muted">admin@eyeson.local</span>
                    </li>
                    <li className="py-2">
                      <strong>Last Login:</strong> <span className="text-muted">{new Date().toLocaleString()}</span>
                    </li>
                  </ul>
                </div>
              </div>
            </div>

            {/* Session Info Card */}
            <div className="col-md-6">
              <div className="card bg-dark border-secondary">
                <div className="card-header border-secondary">
                  <h5 className="mb-0">🔐 Session Information</h5>
                </div>
                <div className="card-body">
                  <ul className="list-unstyled">
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Session Status:</strong> <span className="badge bg-success">Active</span>
                    </li>
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Session Started:</strong> <span className="text-muted">{localStorage.getItem('sessionStart') || new Date().toLocaleString()}</span>
                    </li>
                    <li className="py-2 border-bottom border-secondary">
                      <strong>Token Expires:</strong> <span className="text-warning">In 24 hours</span>
                    </li>
                    <li className="py-2">
                      <strong>API Access:</strong> <span className="badge bg-info">Full Access</span>
                    </li>
                  </ul>
                  <div className="d-grid gap-2 mt-3">
                    <button className="btn btn-outline-warning" onClick={() => {
                      if (window.confirm('Are you sure you want to change your password?')) {
                        alert('Password change feature coming soon!');
                      }
                    }}>
                      🔑 Change Password
                    </button>
                    <button className="btn btn-outline-danger" onClick={handleLogout}>
                      🚪 Logout
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* Preferences Card */}
            <div className="col-md-12">
              <div className="card bg-dark border-secondary">
                <div className="card-header border-secondary">
                  <h5 className="mb-0">⚙️ Preferences</h5>
                </div>
                <div className="card-body">
                  <div className="row g-3">
                    <div className="col-md-4">
                      <label className="form-label text-muted">Theme</label>
                      <div className="d-flex gap-2">
                        <div 
                          className={`theme-option p-3 rounded border ${theme === 'dark' ? 'border-primary bg-primary bg-opacity-25' : 'border-secondary'}`}
                          style={{ cursor: 'pointer', flex: 1, textAlign: 'center' }}
                          onClick={() => setTheme('dark')}
                        >
                          <div style={{ fontSize: '24px', marginBottom: '5px' }}>🌙</div>
                          <div className={theme === 'dark' ? 'text-primary fw-bold' : ''}>Dark</div>
                        </div>
                        <div 
                          className={`theme-option p-3 rounded border ${theme === 'light' ? 'border-primary bg-primary bg-opacity-25' : 'border-secondary'}`}
                          style={{ cursor: 'pointer', flex: 1, textAlign: 'center' }}
                          onClick={() => setTheme('light')}
                        >
                          <div style={{ fontSize: '24px', marginBottom: '5px' }}>☀️</div>
                          <div className={theme === 'light' ? 'text-primary fw-bold' : ''}>Light</div>
                        </div>
                      </div>
                    </div>
                    <div className="col-md-4">
                      <label className="form-label text-muted">Default Page Size</label>
                      <select className="form-select" defaultValue={pagination.limit}>
                        <option value="10">10 items</option>
                        <option value="25">25 items</option>
                        <option value="50">50 items</option>
                        <option value="100">100 items</option>
                      </select>
                    </div>
                    <div className="col-md-4">
                      <label className="form-label text-muted">Language</label>
                      <select className="form-select" defaultValue="en">
                        <option value="en">English</option>
                        <option value="ru">Русский</option>
                      </select>
                    </div>
                  </div>
                  <div className="mt-3">
                    <button className="btn btn-primary" onClick={() => showToast('Preferences saved!', 'success')}>
                      💾 Save Preferences
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      <ToastContainer toasts={toasts} />
    </div>
  );
}

export default App;
