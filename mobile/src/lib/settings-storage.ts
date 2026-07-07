import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

const BASE_URL_KEY = 'newsletter.baseUrl';
// SecureStore keys only allow alphanumerics, ".", "-" and "_".
const API_KEY_KEY = 'newsletter.apiKey';

export async function loadBaseUrl(): Promise<string | null> {
  const stored = await AsyncStorage.getItem(BASE_URL_KEY);
  if (stored) {
    return stored;
  }
  return process.env.EXPO_PUBLIC_API_URL ?? null;
}

export async function saveBaseUrl(baseUrl: string): Promise<void> {
  const trimmed = baseUrl.trim();
  if (trimmed) {
    await AsyncStorage.setItem(BASE_URL_KEY, trimmed);
  } else {
    await AsyncStorage.removeItem(BASE_URL_KEY);
  }
}

// SecureStore is unavailable on web, fall back to AsyncStorage there.
export async function loadApiKey(): Promise<string | null> {
  if (Platform.OS === 'web') {
    return AsyncStorage.getItem(API_KEY_KEY);
  }
  return SecureStore.getItemAsync(API_KEY_KEY);
}

export async function saveApiKey(apiKey: string): Promise<void> {
  const trimmed = apiKey.trim();
  if (Platform.OS === 'web') {
    if (trimmed) {
      await AsyncStorage.setItem(API_KEY_KEY, trimmed);
    } else {
      await AsyncStorage.removeItem(API_KEY_KEY);
    }
    return;
  }
  if (trimmed) {
    await SecureStore.setItemAsync(API_KEY_KEY, trimmed);
  } else {
    await SecureStore.deleteItemAsync(API_KEY_KEY);
  }
}
