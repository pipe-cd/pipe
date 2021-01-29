import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  makeStyles,
  TextField,
  Typography,
} from "@material-ui/core";
import { useFormik } from "formik";
import React, { FC, memo, useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import * as Yup from "yup";
import { UI_TEXT_CANCEL, UI_TEXT_CLOSE } from "../../constants/ui-text";
import { AppState } from "../../modules";
import { Application, selectById } from "../../modules/applications";
import {
  clearSealedSecret,
  generateSealedSecret,
} from "../../modules/sealed-secret";
import { addToast } from "../../modules/toasts";
import { AppDispatch } from "../../store";
import { TextWithCopyButton } from "../text-with-copy-button";

const useStyles = makeStyles((theme) => ({
  targetApp: {
    color: theme.palette.text.primary,
    fontWeight: theme.typography.fontWeightMedium,
  },
  secretInput: {
    border: "none",
    fontSize: 14,
    flex: 1,
    textOverflow: "ellipsis",
  },
  encryptedSecret: {
    wordBreak: "break-all",
  },
}));

interface Props {
  applicationId: string | null;
  open: boolean;
  onClose: () => void;
}

const DIALOG_TITLE = "Encrypting secret data for application";
const BASE64_CHECKBOX_LABEL =
  "Use base64 encoding before encrypting the secret";

const validationSchema = Yup.object({
  secretData: Yup.string().required(),
  base64: Yup.bool(),
});

export const SealedSecretDialog: FC<Props> = memo(function SealedSecretDialog({
  open,
  applicationId,
  onClose,
}) {
  const classes = useStyles();
  const dispatch = useDispatch<AppDispatch>();

  const [application, isLoading, sealedSecret] = useSelector<
    AppState,
    [Application.AsObject | undefined, boolean, string | null]
  >((state) => [
    applicationId ? selectById(state.applications, applicationId) : undefined,
    state.sealedSecret.isLoading,
    state.sealedSecret.data,
  ]);

  const formik = useFormik({
    initialValues: {
      secretData: "",
      base64: false,
    },
    validationSchema,
    validateOnMount: true,
    async onSubmit(values) {
      if (!application) {
        return;
      }
      await dispatch(
        generateSealedSecret({
          data: values.secretData,
          pipedId: application.pipedId,
          base64Encoding: values.base64,
        })
      );
    },
  });

  const handleOnEnter = useCallback(() => {
    formik.resetForm();
  }, [formik]);

  const handleOnExited = (): void => {
    // Clear state after closed dialog
    setTimeout(() => {
      dispatch(clearSealedSecret());
    }, 200);
  };

  const handleOnClickCopy = useCallback(() => {
    dispatch(addToast({ message: "Secret copied to clipboard" }));
  }, [dispatch]);

  if (!application) {
    return null;
  }

  return (
    <Dialog
      open={open}
      onEnter={handleOnEnter}
      onExit={handleOnExited}
      onClose={onClose}
    >
      {sealedSecret ? (
        <>
          <DialogTitle>{DIALOG_TITLE}</DialogTitle>
          <DialogContent>
            <Typography variant="caption" color="textSecondary">
              Encrypted secret data
            </Typography>
            <TextWithCopyButton
              label="Copy secret"
              value={sealedSecret}
              onCopy={handleOnClickCopy}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={onClose}>{UI_TEXT_CLOSE}</Button>
          </DialogActions>
        </>
      ) : (
        <form onSubmit={formik.handleSubmit}>
          <DialogTitle>{DIALOG_TITLE}</DialogTitle>
          <DialogContent>
            <Typography variant="caption" color="textSecondary">
              Application
            </Typography>
            <Typography variant="body1" className={classes.targetApp}>
              {application.name}
            </Typography>
            <TextField
              id="secretData"
              name="secretData"
              value={formik.values.secretData}
              variant="outlined"
              margin="dense"
              label="Secret Data"
              multiline
              fullWidth
              rows={4}
              required
              autoFocus
              onChange={formik.handleChange}
            />
            <FormControlLabel
              control={
                <Checkbox
                  color="primary"
                  checked={formik.values.base64}
                  onChange={formik.handleChange}
                  name="base64"
                />
              }
              disabled={formik.isSubmitting}
              label={BASE64_CHECKBOX_LABEL}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={onClose} disabled={isLoading}>
              {UI_TEXT_CANCEL}
            </Button>
            <Button
              color="primary"
              type="submit"
              disabled={isLoading || formik.isValid === false}
            >
              Encrypt
            </Button>
          </DialogActions>
        </form>
      )}
    </Dialog>
  );
});
